package aggregate

import (
	"testing"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
	"github.com/vutratenko/shturval-alert-wall/internal/mood"
)

func TestGroup_sameNodeWithinWindow(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	alerts := []model.Alert{
		nodeAlert("n", "KubeNodeNotReady", model.BucketCritical, "worker-04", base),
		nodeAlert("p1", "KubePodNotReady", model.BucketWarning, "worker-04", base.Add(5*time.Minute)),
		nodeAlert("p2", "KubePodCrashLooping", model.BucketWarning, "worker-04", base.Add(15*time.Minute)),
	}
	groups := Group("demo", alerts, 15*time.Minute)
	if len(groups) != 1 {
		t.Fatalf("groups %d", len(groups))
	}
	g := groups[0]
	if g.KeyKind != "node" || g.KeyValue != "worker-04" || g.Severity != model.BucketCritical {
		t.Fatalf("%+v", g)
	}
	if g.Root.Fingerprint != "n" || len(g.Members) != 3 || g.Members[0].Fingerprint != "n" {
		t.Fatalf("root %+v members %d", g.Root, len(g.Members))
	}
	snap := buildDemo(alerts, nil, 15*time.Minute)
	if snap.Clusters[0].Counts.Critical != 1 || snap.Clusters[0].Counts.Warning != 0 || snap.Mood != model.MoodWorried {
		t.Fatalf("counts %+v mood %s", snap.Clusters[0].Counts, snap.Mood)
	}
}

func TestGroup_splitByGapAndNode(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	alerts := []model.Alert{
		nodeAlert("a", "A", model.BucketCritical, "worker-04", base),
		nodeAlert("b", "B", model.BucketCritical, "worker-04", base.Add(20*time.Minute)),
		nodeAlert("c", "C", model.BucketCritical, "worker-05", base),
	}
	groups := Group("demo", alerts, 15*time.Minute)
	if len(groups) != 3 {
		t.Fatalf("groups %d", len(groups))
	}
	snap := buildDemo(alerts, nil, 15*time.Minute)
	if snap.Clusters[0].Counts.Critical != 3 || snap.Mood != model.MoodPanic {
		t.Fatalf("counts %+v mood %s", snap.Clusters[0].Counts, snap.Mood)
	}
}

func TestBuild_separateNodesPanic(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	alerts := []model.Alert{
		nodeAlert("a", "A", model.BucketCritical, "worker-04", base),
		nodeAlert("b", "B", model.BucketCritical, "worker-05", base),
	}
	snap := buildDemo(alerts, nil, 15*time.Minute)
	if snap.Clusters[0].Counts.Critical != 2 || snap.Mood != model.MoodPanic {
		t.Fatalf("counts %+v mood %s", snap.Clusters[0].Counts, snap.Mood)
	}
}

func TestGroup_podAndWorkloadWithoutNode(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	pod := []model.Alert{
		{Fingerprint: "p1", Alertname: "Crash", Severity: model.BucketWarning, Namespace: "orders", StartsAt: base,
			Labels: map[string]string{"pod": "checkout", "node": "worker-04"}},
		{Fingerprint: "p2", Alertname: "NotReady", Severity: model.BucketWarning, Namespace: "orders", StartsAt: base.Add(time.Minute),
			Labels: map[string]string{"pod": "checkout"}},
		{Fingerprint: "p3", Alertname: "Waiting", Severity: model.BucketInfo, Namespace: "orders", StartsAt: base.Add(2 * time.Minute),
			Labels: map[string]string{"pod": "checkout"}},
	}
	groups := Group("demo", pod, 15*time.Minute)
	if len(groups) != 2 {
		t.Fatalf("pod groups %d", len(groups))
	}
	var podGroup, nodeGroup int
	for _, g := range groups {
		switch g.KeyKind {
		case "pod":
			podGroup++
			if g.KeyValue != "orders/checkout" || len(g.Members) != 2 {
				t.Fatalf("%+v", g)
			}
		case "node":
			nodeGroup++
		default:
			t.Fatalf("kind %s", g.KeyKind)
		}
	}
	if podGroup != 1 || nodeGroup != 1 {
		t.Fatalf("pod %d node %d", podGroup, nodeGroup)
	}

	workload := []model.Alert{
		{Fingerprint: "w1", Alertname: "Replicas", Severity: model.BucketCritical, Namespace: "ml", StartsAt: base,
			Labels: map[string]string{"statefulset": "training-worker"}},
		{Fingerprint: "w2", Alertname: "Slow", Severity: model.BucketWarning, Namespace: "ml", StartsAt: base.Add(time.Minute),
			Labels: map[string]string{"statefulset": "training-worker"}},
	}
	got := Group("demo", workload, 15*time.Minute)
	if len(got) != 1 || got[0].KeyKind != "workload" || got[0].KeyValue != "ml/statefulset/training-worker" {
		t.Fatalf("%+v", got)
	}
}

func TestGroup_singletonAndSilenced(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	alerts := []model.Alert{{Fingerprint: "solo", Alertname: "DiskFull", Severity: model.BucketCritical, StartsAt: base}}
	groups := Group("demo", alerts, 15*time.Minute)
	if len(groups) != 1 || groups[0].KeyKind != "alert" || groups[0].KeyValue != "solo" {
		t.Fatalf("%+v", groups)
	}
	silenced := []model.SilencedAlert{{Alert: model.Alert{Fingerprint: "s", Severity: model.BucketCritical}}}
	snap := buildDemo(nil, silenced, 15*time.Minute)
	if len(snap.Clusters[0].Groups) != 0 || snap.Clusters[0].Counts.Critical != 0 {
		t.Fatalf("%+v", snap.Clusters[0])
	}
}

func TestBuild_windowZeroCountsAlerts(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	alerts := []model.Alert{
		nodeAlert("a", "A", model.BucketCritical, "worker-04", base),
		nodeAlert("b", "B", model.BucketCritical, "worker-04", base),
	}
	snap := buildDemo(alerts, nil, 0)
	if len(snap.Clusters[0].Groups) != 0 || snap.Clusters[0].Counts.Critical != 2 || snap.Mood != model.MoodPanic {
		t.Fatalf("counts %+v groups %d mood %s", snap.Clusters[0].Counts, len(snap.Clusters[0].Groups), snap.Mood)
	}
}

func TestGroup_namespaceDoesNotMerge(t *testing.T) {
	base := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	alerts := []model.Alert{
		{Fingerprint: "a", Alertname: "A", Severity: model.BucketWarning, Namespace: "kube-system", StartsAt: base,
			Labels: map[string]string{"pod": "a"}},
		{Fingerprint: "b", Alertname: "B", Severity: model.BucketWarning, Namespace: "kube-system", StartsAt: base,
			Labels: map[string]string{"pod": "b"}},
	}
	if len(Group("demo", alerts, 15*time.Minute)) != 2 {
		t.Fatal("namespace grouped independent pods")
	}
}

func nodeAlert(fp, name string, sev model.Bucket, node string, at time.Time) model.Alert {
	return model.Alert{
		Fingerprint: fp, Alertname: name, Severity: sev, Namespace: "ns", StartsAt: at,
		Labels: map[string]string{"node": node},
	}
}

func buildDemo(alerts []model.Alert, silenced []model.SilencedAlert, window time.Duration) model.Snapshot {
	return Build([]ClusterFetch{{
		Name: "demo", Reachable: true, Alerts: alerts, Silenced: silenced,
	}}, time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC), mood.DefaultThresholds(), window)
}
