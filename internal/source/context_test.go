package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

func TestContextQueries_quotesLabelAndSkipsBareCluster(t *testing.T) {
	alert := model.Alert{
		Alertname: "KubeNodeNotReady",
		Cluster:   "victoria-metrics",
		Namespace: "orders",
		Labels:    map[string]string{"node": `worker"04`, "pod": "checkout", "cluster": "prod"},
	}
	queries := ContextQueries(alert)
	if len(queries) != 2 {
		t.Fatalf("%d queries", len(queries))
	}
	if !strings.Contains(queries[0].Query, `node="worker\"04"`) || !strings.Contains(queries[0].Query, `cluster="prod"`) {
		t.Fatalf("node query %s", queries[0].Query)
	}
	if queries[0].Name != "нода Ready" || queries[1].Name != "рестарты пода" {
		t.Fatalf("%+v", queries)
	}
	if !strings.Contains(queries[1].Query, `namespace="orders"`) || !strings.Contains(queries[1].Query, `pod="checkout"`) {
		t.Fatalf("pod query %s", queries[1].Query)
	}

	bare := model.Alert{Cluster: "victoria-metrics", Labels: map[string]string{"node": "worker-04"}}
	got := ContextQueries(bare)
	if len(got) != 1 || strings.Contains(got[0].Query, "cluster=") {
		t.Fatalf("bare cluster leaked into %s", got[0].Query)
	}
}

func TestContextQueries_emptyWithoutNodeOrPod(t *testing.T) {
	if len(ContextQueries(model.Alert{Alertname: "DiskFull"})) != 0 {
		t.Fatal("expected no queries")
	}
}

func TestQueryContext_rangeAndNoCall(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if !strings.HasSuffix(r.URL.Path, "/api/v1/query_range") {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("step") != "60" {
			t.Errorf("step %s", r.URL.Query().Get("step"))
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"values":[[100,"1"],[160,"0"]]}]}}`))
	}))
	defer ts.Close()

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	alert := model.Alert{
		StartsAt: now.Add(-time.Hour),
		Labels:   map[string]string{"node": "worker-04"},
	}
	got, err := QueryContext(context.Background(), ts.URL+"/select/0/prometheus", ts.Client(), alert, now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Available || len(got.Samples) != 1 || got.Samples[0].Before != "1" || got.Samples[0].After != "0" {
		t.Fatalf("%+v", got)
	}
	if hits != 1 {
		t.Fatalf("hits %d", hits)
	}

	hits = 0
	plain, err := QueryContext(context.Background(), ts.URL, ts.Client(), model.Alert{Alertname: "DiskFull"}, now)
	if err != nil || !plain.Available || len(plain.Samples) != 0 || hits != 0 {
		t.Fatalf("%+v err %v hits %d", plain, err, hits)
	}
}

func TestFormatSample(t *testing.T) {
	if formatSample(0) != "0" || formatSample(1) != "1" || formatSample(3) != "3.0" || formatSample(1.25) != "1.3" {
		t.Fatalf("%s %s %s %s", formatSample(0), formatSample(1), formatSample(3), formatSample(1.25))
	}
}
