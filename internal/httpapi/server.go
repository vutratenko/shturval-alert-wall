package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vutratenko/shturval-alert-wall/internal/auth"
	"github.com/vutratenko/shturval-alert-wall/internal/push"
)

func init() {
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

type Server struct {
	Hub    *Hub
	Static fs.FS
	Push   *push.Store
}

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok", "service": "shturval-alert-wall"})
	})
	m.HandleFunc("GET /api/snapshot", s.snapshot)
	m.HandleFunc("GET /api/stream", s.stream)
	m.HandleFunc("GET /api/push/vapid", s.pushVapid)
	m.HandleFunc("PUT /api/push/subscription", s.putSubscription)
	m.HandleFunc("DELETE /api/push/subscription", s.deleteSubscription)
	m.Handle("GET /metrics", promhttp.Handler())
	if s.Static != nil {
		m.Handle("GET /wall", spa(s.Static))
		m.Handle("GET /wall/", spa(s.Static))
		m.Handle("/", spa(s.Static))
	}
	return m
}

func (s *Server) snapshot(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, s.Hub.Snapshot())
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	snap := s.Hub.Snapshot()
	writeSSE(w, flusher, "snapshot", snap)

	ch := s.Hub.Subscribe(32)
	defer s.Hub.Unsubscribe(ch)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, open := <-ch:
			if !open {
				return
			}
			writeSSE(w, flusher, ev.Type, ev)
		case <-ticker.C:
			_, _ = fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	flusher.Flush()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func cacheControl(path string) string {
	if path == "/" || strings.HasSuffix(path, ".html") || path == "/sw.js" || path == "/manifest.webmanifest" {
		return "no-cache"
	}
	if strings.HasPrefix(path, "/assets/") {
		return "public, max-age=31536000, immutable"
	}
	return ""
}

func (s *Server) pushVapid(w http.ResponseWriter, r *http.Request) {
	if s.Push == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"publicKey": s.Push.PublicKey()})
}

func (s *Server) putSubscription(w http.ResponseWriter, r *http.Request) {
	if s.Push == nil {
		http.NotFound(w, r)
		return
	}
	sub, ok := decodeSubscription(w, r)
	if !ok {
		return
	}
	sub.Sub = auth.Subject(r.Context())
	if err := s.Push.Save(sub); err != nil {
		if errors.Is(err, push.ErrInvalid) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	if s.Push == nil {
		http.NotFound(w, r)
		return
	}
	sub, ok := decodeSubscription(w, r)
	if !ok {
		return
	}
	if err := s.Push.Delete(sub.Endpoint); err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeSubscription(w http.ResponseWriter, r *http.Request) (push.Subscription, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<14)
	var sub push.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil || strings.TrimSpace(sub.Endpoint) == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return push.Subscription{}, false
	}
	return sub, true
}

func spa(root fs.FS) http.Handler {
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" && p != "wall" && !strings.HasPrefix(p, "wall/") {
			if _, err := fs.Stat(root, p); err != nil {
				r.URL.Path = "/"
			}
		}
		if strings.HasPrefix(p, "wall") {
			r.URL.Path = "/"
		}
		if cc := cacheControl(r.URL.Path); cc != "" {
			w.Header().Set("Cache-Control", cc)
		}
		files.ServeHTTP(w, r)
	})
}
