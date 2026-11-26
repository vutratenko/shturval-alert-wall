package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

func TestAlertmanager_Fetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/alerts" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"fingerprint":  "fp1",
				"labels":       map[string]string{"alertname": "KubePodCrashLooping", "severity": "warning", "namespace": "orders", "pod": "api-7f9c"},
				"annotations":  map[string]string{"summary": "pod down"},
				"generatorURL": "http://prometheus/graph?g0.expr=up",
				"startsAt":     time.Now().UTC().Format(time.RFC3339Nano),
			},
		})
	}))
	defer ts.Close()
	am := &Alertmanager{ClusterName: "test", BaseURL: ts.URL}
	batch, err := am.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	alerts := batch.Alerts
	if len(alerts) != 1 || alerts[0].Alertname != "KubePodCrashLooping" {
		t.Fatalf("%+v", alerts)
	}
	if alerts[0].Labels["pod"] != "api-7f9c" {
		t.Fatalf("labels: %+v", alerts[0].Labels)
	}
	if alerts[0].GeneratorURL != "http://prometheus/graph?g0.expr=up" {
		t.Fatalf("generator: %q", alerts[0].GeneratorURL)
	}
}

func TestAlertmanager_Fetch_500(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer ts.Close()
	am := &Alertmanager{ClusterName: "test", BaseURL: ts.URL}
	_, err := am.Fetch(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAlertmanager_FetchSplitsSilenced(t *testing.T) {
	ends := time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/alerts":
			if r.URL.Query().Get("active") != "true" || r.URL.Query().Get("silenced") != "true" || r.URL.Query().Get("inhibited") != "false" {
				t.Errorf("query %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"fingerprint": "fp-hot",
					"labels":      map[string]string{"alertname": "Hot", "severity": "critical"},
					"startsAt":    time.Now().UTC().Format(time.RFC3339Nano),
				},
				{
					"fingerprint": "fp-quiet",
					"labels":      map[string]string{"alertname": "Quiet", "severity": "warning", "namespace": "orders"},
					"startsAt":    time.Now().UTC().Format(time.RFC3339Nano),
					"status":      map[string]any{"state": "suppressed", "silencedBy": []string{"sil-1"}},
				},
				{
					"fingerprint": "fp-inhib",
					"labels":      map[string]string{"alertname": "Hidden", "severity": "info"},
					"startsAt":    time.Now().UTC().Format(time.RFC3339Nano),
					"status":      map[string]any{"silencedBy": []string{"sil-1"}, "inhibitedBy": []string{"inh-1"}},
				},
			})
		case "/api/v2/silences":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"id":        "sil-1",
					"status":    map[string]string{"state": "active"},
					"comment":   "maintenance",
					"createdBy": "a@b.c",
					"endsAt":    ends,
					"matchers": []map[string]any{
						{"name": "alertname", "value": "Quiet", "isRegex": false, "isEqual": true},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	batch, err := (&Alertmanager{ClusterName: "test", BaseURL: ts.URL}).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Alerts) != 1 || batch.Alerts[0].Fingerprint != "fp-hot" {
		t.Fatalf("alerts %+v", batch.Alerts)
	}
	if len(batch.Silenced) != 1 || batch.Silenced[0].Fingerprint != "fp-quiet" {
		t.Fatalf("silenced %+v", batch.Silenced)
	}
	refs := batch.Silenced[0].Silences
	if len(refs) != 1 || refs[0].ID != "sil-1" || refs[0].CreatedBy != "a@b.c" || refs[0].Comment != "maintenance" {
		t.Fatalf("refs %+v", refs)
	}
}

func TestAlertmanager_CreateSilenceReusesMatchers(t *testing.T) {
	var posts int
	matchers := []model.Matcher{{Name: "alertname", Value: "Quiet", IsEqual: true}}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/silences":
			if posts == 0 {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id":        "sil-1",
				"status":    map[string]string{"state": "active"},
				"endsAt":    time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano),
				"createdBy": "a@b.c",
				"comment":   "hold",
				"matchers":  matchers,
			}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/silences":
			posts++
			var body struct {
				Matchers  []model.Matcher `json:"matchers"`
				CreatedBy string          `json:"createdBy"`
				Comment   string          `json:"comment"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.CreatedBy != "a@b.c" || body.Comment != "hold" || !model.SameMatchers(body.Matchers, matchers) {
				t.Fatalf("%+v", body)
			}
			for _, m := range body.Matchers {
				if m.IsRegex || !m.IsEqual {
					t.Fatalf("matcher %+v", m)
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"silenceID": "sil-1"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	am := &Alertmanager{ClusterName: "test", BaseURL: ts.URL}
	draft := model.SilenceDraft{
		Matchers:  matchers,
		StartsAt:  time.Now().UTC(),
		EndsAt:    time.Now().UTC().Add(time.Hour),
		CreatedBy: "a@b.c",
		Comment:   "hold",
	}
	first, err := am.CreateSilence(context.Background(), draft)
	if err != nil || first.ID != "sil-1" {
		t.Fatalf("%v %+v", err, first)
	}
	second, err := am.CreateSilence(context.Background(), draft)
	if err != nil || second.ID != "sil-1" || posts != 1 {
		t.Fatalf("posts %d err %v %+v", posts, err, second)
	}
}

func TestAlertmanager_ExpireSilence(t *testing.T) {
	var path string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.NotFound(w, r)
			return
		}
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	err := (&Alertmanager{ClusterName: "test", BaseURL: ts.URL}).ExpireSilence(context.Background(), "sil-1")
	if err != nil || path != "/api/v2/silence/sil-1" {
		t.Fatalf("%v %s", err, path)
	}
}
