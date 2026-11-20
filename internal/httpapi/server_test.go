package httpapi

import (
	"bufio"
	"context"
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
	srv := &Server{Hub: hub}
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/stream", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.stream(rec, req)
	}()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			cancel()
			t.Fatal("timeout")
		default:
			if strings.Contains(rec.Body.String(), "event: snapshot") {
				cancel()
				<-done
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
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
	sc := bufio.NewScanner(strings.NewReader(rec.Body.String()))
	if !sc.Scan() {
		t.Fatal("empty body")
	}
	if !strings.Contains(sc.Text(), `"mood"`) {
		t.Fatal(rec.Body.String())
	}
}