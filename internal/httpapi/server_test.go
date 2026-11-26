package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/auth"
	"github.com/vutratenko/shturval-alert-wall/internal/model"
	"github.com/vutratenko/shturval-alert-wall/internal/push"
)

func TestStream_sendsSnapshot(t *testing.T) {
	hub := NewHub()
	hub.Update(model.Snapshot{
		GeneratedAt: time.Now().UTC(),
		Mood:        model.MoodCalm,
		Clusters:    []model.ClusterSummary{},
	})
	srv := httptest.NewServer((&Server{Hub: hub}).Handler())
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	var found bool
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "event: snapshot") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no snapshot event")
	}
}

func TestHealthz(t *testing.T) {
	srv := &Server{Hub: NewHub()}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/healthz", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" || body["service"] != "shturval-alert-wall" {
		t.Fatalf("%v", body)
	}
}

func TestSPA_DocumentNoCache(t *testing.T) {
	root := fstest.MapFS{
		"index.html":           {Data: []byte("<!doctype html><title>wall</title>")},
		"assets/app.css":       {Data: []byte("body{}")},
		"sw.js":                {Data: []byte("self.addEventListener('fetch',()=>{})")},
		"manifest.webmanifest": {Data: []byte(`{"name":"Пульт"}`)},
	}
	srv := &Server{Hub: NewHub(), Static: root}

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wall", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("document cache-control %q", cc)
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.css", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("asset cache-control %q", cc)
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sw.js", nil))
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("sw %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("sw content-type %q", ct)
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil))
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("manifest %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/manifest+json") {
		t.Fatalf("manifest content-type %q", ct)
	}
}

func TestSnapshotJSON(t *testing.T) {
	hub := NewHub()
	srv := &Server{Hub: hub}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/snapshot", nil))
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"mood"`) {
		t.Fatal(string(body))
	}
}

func TestPushSubscription_roundTrip(t *testing.T) {
	store, err := push.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{Hub: NewHub(), Push: store}
	body := `{"endpoint":" https://push.example/abc ","keys":{"p256dh":"k","auth":"a"},"sub":"spoof"}`
	req := httptest.NewRequest(http.MethodPut, "/api/push/subscription", strings.NewReader(body))
	req = req.WithContext(auth.WithSession(req.Context(), "user-1", "a@b.c"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("put %d %s", rec.Code, rec.Body.String())
	}
	subs := store.List()
	if len(subs) != 1 || subs[0].Sub != "user-1" || subs[0].Endpoint != "https://push.example/abc" {
		t.Fatalf("%+v", subs)
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/push/vapid", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), store.PublicKey()) {
		t.Fatalf("vapid %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	del := httptest.NewRequest(http.MethodDelete, "/api/push/subscription", strings.NewReader(`{"endpoint":"https://push.example/abc"}`))
	srv.Handler().ServeHTTP(rec, del)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete %d", rec.Code)
	}
	if len(store.List()) != 0 {
		t.Fatal("subscription remained")
	}
}

func TestPushSubscription_missingStore(t *testing.T) {
	srv := &Server{Hub: NewHub()}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/push/vapid", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatal(rec.Code)
	}
}
