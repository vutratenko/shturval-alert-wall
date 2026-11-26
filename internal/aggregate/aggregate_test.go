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
	}, now, mood.DefaultThresholds(), 0, "")
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
	}}, time.Now().UTC(), mood.DefaultThresholds(), 0, "")
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

func TestDiff_groupCascade(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	var first []model.Alert
	for i := 0; i < 6; i++ {
		first = append(first, nodeAlert(
			"a"+string(rune('0'+i)), "Pod", model.BucketWarning, "worker-04", base.Add(time.Duration(i)*time.Minute),
		))
	}
	first[0].Alertname = "KubeNodeNotReady"
	prev := buildDemo(nil, nil, 15*time.Minute)
	opened := Diff(prev, buildDemo(first, nil, 15*time.Minute))
	if countEvents(opened, "group.opened") != 1 || countEvents(opened, "alert.firing") != 0 {
		t.Fatalf("%+v", opened)
	}
	if opened[0].Group == nil || opened[0].Alert == nil || opened[0].Alert.Alertname != "KubeNodeNotReady" {
		t.Fatalf("%+v", opened)
	}

	grown := append(first, nodeAlert("extra", "KubePodNotReady", model.BucketWarning, "worker-04", base.Add(6*time.Minute)))
	quiet := Diff(buildDemo(first, nil, 15*time.Minute), buildDemo(grown, nil, 15*time.Minute))
	if countEvents(quiet, "group.opened")+countEvents(quiet, "group.promoted")+countEvents(quiet, "alert.firing") != 0 {
		t.Fatalf("%+v", quiet)
	}

	promotedAlerts := append([]model.Alert{}, first...)
	promotedAlerts = append(promotedAlerts, nodeAlert("crit", "DiskFull", model.BucketCritical, "worker-04", base.Add(7*time.Minute)))
	promoted := Diff(buildDemo(first, nil, 15*time.Minute), buildDemo(promotedAlerts, nil, 15*time.Minute))
	if countEvents(promoted, "group.promoted") != 1 || countEvents(promoted, "group.opened") != 0 {
		t.Fatalf("%+v", promoted)
	}
}

func TestDiff_anchorLeavesSymptoms(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	anchor := nodeAlert("n", "KubeNodeNotReady", model.BucketCritical, "worker-04", base)
	symptom := nodeAlert("p", "KubePodNotReady", model.BucketCritical, "worker-04", base.Add(time.Minute))
	events := Diff(buildDemo([]model.Alert{anchor, symptom}, nil, 15*time.Minute), buildDemo([]model.Alert{symptom}, nil, 15*time.Minute))
	if countEvents(events, "group.opened")+countEvents(events, "group.resolved")+countEvents(events, "alert.firing")+countEvents(events, "alert.resolved") != 0 {
		t.Fatalf("%+v", events)
	}
}

func TestDiff_silencedGroupStaysOpen(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	alerts := []model.Alert{
		nodeAlert("n", "KubeNodeNotReady", model.BucketCritical, "worker-04", base),
		nodeAlert("p", "KubePodNotReady", model.BucketWarning, "worker-04", base.Add(time.Minute)),
	}
	silenced := []model.SilencedAlert{{Alert: alerts[0]}, {Alert: alerts[1]}}
	events := Diff(buildDemo(alerts, nil, 15*time.Minute), buildDemo(nil, silenced, 15*time.Minute))
	if countEvents(events, "group.resolved") != 0 || countEvents(events, "alert.resolved") != 0 || countEvents(events, "alert.silenced") != 2 {
		t.Fatalf("%+v", events)
	}
}

func hasEvent(events []Event, kind string) bool {
	return countEvents(events, kind) > 0
}

func countEvents(events []Event, kind string) int {
	n := 0
	for _, e := range events {
		if e.Type == kind {
			n++
		}
	}
	return n
}
