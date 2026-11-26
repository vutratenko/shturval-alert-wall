package aggregate

import (
	"testing"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
	"github.com/vutratenko/shturval-alert-wall/internal/mood"
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

func TestBuild_ignoresSilencedInCounts(t *testing.T) {
	snap := Build([]ClusterFetch{{
		Name:      "a",
		Source:    "am",
		Reachable: true,
		Silenced: []model.SilencedAlert{{
			Alert: model.Alert{Fingerprint: "1", Severity: model.BucketCritical},
		}},
	}}, time.Now().UTC(), mood.DefaultThresholds())
	if snap.Clusters[0].Counts.Critical != 0 || snap.Mood != model.MoodCalm {
		t.Fatalf("%+v mood %s", snap.Clusters[0].Counts, snap.Mood)
	}
	if snap.Clusters[0].Source != "am" || len(snap.Clusters[0].Silenced) != 1 {
		t.Fatalf("%+v", snap.Clusters[0])
	}
}

func TestDiff_silenceIsNotResolved(t *testing.T) {
	now := time.Now().UTC()
	alert := model.Alert{Fingerprint: "x", Alertname: "A"}
	prev := model.Snapshot{GeneratedAt: now, Clusters: []model.ClusterSummary{{
		Name: "c", Reachable: true, Alerts: []model.Alert{alert},
	}}}
	next := model.Snapshot{GeneratedAt: now.Add(time.Second), Clusters: []model.ClusterSummary{{
		Name: "c", Reachable: true, Silenced: []model.SilencedAlert{{Alert: alert}},
	}}}
	events := Diff(prev, next)
	if hasEvent(events, "alert.resolved") || hasEvent(events, "alert.firing") || !hasEvent(events, "alert.silenced") {
		t.Fatalf("%+v", events)
	}

	back := Diff(next, prev)
	if !hasEvent(back, "alert.unsilenced") || !hasEvent(back, "alert.firing") || hasEvent(back, "alert.resolved") {
		t.Fatalf("%+v", back)
	}

	gone := model.Snapshot{GeneratedAt: now.Add(2 * time.Second), Clusters: []model.ClusterSummary{{
		Name: "c", Reachable: true,
	}}}
	if !hasEvent(Diff(next, gone), "alert.resolved") {
		t.Fatal("silenced alert disappeared without resolved")
	}
}

func TestDiff_newSilencedClusterStaysQuiet(t *testing.T) {
	now := time.Now().UTC()
	next := model.Snapshot{GeneratedAt: now, Clusters: []model.ClusterSummary{{
		Name: "c", Reachable: true,
		Silenced: []model.SilencedAlert{{Alert: model.Alert{Fingerprint: "x", Alertname: "A"}}},
	}}}
	if events := Diff(model.Snapshot{GeneratedAt: now.Add(-time.Second)}, next); len(events) != 0 {
		t.Fatalf("%+v", events)
	}
}

func hasEvent(events []Event, kind string) bool {
	for _, e := range events {
		if e.Type == kind {
			return true
		}
	}
	return false
}
