package httpapi

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
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