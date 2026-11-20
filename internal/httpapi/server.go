package httpapi

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	Hub    *Hub
	Static fs.FS
}

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	m.HandleFunc("GET /api/snapshot", s.snapshot)
	m.HandleFunc("GET /api/stream", s.stream)
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
		files.ServeHTTP(w, r)
	})
}