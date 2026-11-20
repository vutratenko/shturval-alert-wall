package aggregate

import (
	"testing"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/mood"
	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

func TestBuild_sortsBySeverity(t *testing.T) {
	now := time.Now().UTC()
	snap := Build([]ClusterFetch{
		{Name: "a", Reachable: true, Alerts: []model.Alert{{Fingerprint: "1", Severity: model.BucketInfo}}},
		{Name: "b", Reachable: true, Alerts: []model.Alert{{Fingerprint: "2", Severity: model.BucketCritical}}},
	}, now, mood.DefaultThresholds())
	if snap.Clusters[0].Name != "b" {
		t.Fatalf("got %q first", snap.Clusters[0].Name)
	}
	if snap.Mood != model.MoodWorried {
		t.Fatalf("mood %q", snap.Mood)
	}
}

func TestDiff_firingAndResolved(t *testing.T) {
	now := time.Now().UTC()
	prev := model.Snapshot{
		GeneratedAt: now,
		Clusters: []model.ClusterSummary{
			{Name: "c", Reachable: true, Alerts: []model.Alert{{Fingerprint: "x", Alertname: "A"}}},
		},
	}
	next := model.Snapshot{
		GeneratedAt: now.Add(time.Second),
		Clusters: []model.ClusterSummary{
			{
				Name: "c", Reachable: true,
				Alerts: []model.Alert{
					{Fingerprint: "y", Alertname: "B"},
				},
			},
		},
	}
	events := Diff(prev, next)
	var hasFiring, hasResolved bool
	for _, e := range events {
		if e.Type == "alert.firing" && e.Alert != nil && e.Alert.Alertname == "B" {
			hasFiring = true
		}
		if e.Type == "alert.resolved" && e.Alert != nil && e.Alert.Alertname == "A" {
			hasResolved = true
		}
	}
	if !hasFiring || !hasResolved {
		t.Fatalf("events %+v", events)
	}
}