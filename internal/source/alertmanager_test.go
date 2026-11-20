package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAlertmanager_Fetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/alerts" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"fingerprint": "fp1",
				"labels":      map[string]string{"alertname": "KubePodCrashLooping", "severity": "warning", "namespace": "orders"},
				"annotations": map[string]string{"summary": "pod down"},
				"startsAt":    time.Now().UTC().Format(time.RFC3339Nano),
			},
		})
	}))
	defer ts.Close()
	am := &Alertmanager{ClusterName: "test", BaseURL: ts.URL}
	alerts, err := am.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Alertname != "KubePodCrashLooping" {
		t.Fatalf("%+v", alerts)
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