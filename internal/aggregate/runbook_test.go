package aggregate

import (
	"testing"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
	"github.com/vutratenko/shturval-alert-wall/internal/mood"
)

func TestBuild_fillsMissingRunbook(t *testing.T) {
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	snap := Build([]ClusterFetch{{
		Name: "demo", Reachable: true,
		Alerts: []model.Alert{
			{Fingerprint: "a", Alertname: "KubeNodeNotReady", Severity: model.BucketCritical},
			{Fingerprint: "b", Alertname: "DiskFull", Severity: model.BucketWarning, RunbookURL: "https://keep.example/disk"},
			{Fingerprint: "c", Alertname: "  ", Severity: model.BucketInfo},
		},
		Silenced: []model.SilencedAlert{{
			Alert: model.Alert{Fingerprint: "s", Alertname: "KubePodNotReady", Severity: model.BucketWarning},
		}},
	}}, now, mood.DefaultThresholds(), 0, "https://runbooks.example/")
	got := map[string]string{}
	for _, alert := range snap.Clusters[0].Alerts {
		got[alert.Fingerprint] = alert.RunbookURL
	}
	if got["a"] != "https://runbooks.example/KubeNodeNotReady" {
		t.Fatalf("filled %q", got["a"])
	}
	if got["b"] != "https://keep.example/disk" {
		t.Fatalf("explicit %q", got["b"])
	}
	if got["c"] != "" {
		t.Fatalf("blank name %q", got["c"])
	}
	if snap.Clusters[0].Silenced[0].RunbookURL != "https://runbooks.example/KubePodNotReady" {
		t.Fatalf("silenced %q", snap.Clusters[0].Silenced[0].RunbookURL)
	}
}

func TestRunbookLink_rejectsUnsafeBase(t *testing.T) {
	if runbookLink("", "javascript:alert(1)", "KubeNodeNotReady") != "" {
		t.Fatal("javascript base became a link")
	}
	if runbookLink("", "", "KubeNodeNotReady") != "" {
		t.Fatal("empty base became a link")
	}
}
