package push

import (
	"testing"

	"github.com/vutratenko/shturval-alert-wall/internal/aggregate"
	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

func TestMessage_singleFiring(t *testing.T) {
	title, body, ok := Message([]aggregate.Event{
		{Type: "alert.resolved", Cluster: "old", Alert: &model.Alert{Alertname: "Gone"}},
		{Type: "alert.firing", Cluster: "prod", Alert: &model.Alert{Alertname: "DiskFull"}},
	})
	if !ok || title != "Штурвал" || body != "DiskFull · prod" {
		t.Fatalf("%q %q %v", title, body, ok)
	}
}

func TestMessage_coalescesABurst(t *testing.T) {
	title, body, ok := Message([]aggregate.Event{
		{Type: "alert.firing", Cluster: "a", Alert: &model.Alert{Alertname: "A"}},
		{Type: "alert.firing", Cluster: "b", Alert: &model.Alert{Alertname: "B"}},
	})
	if !ok || title != "Штурвал" || body != "2 новых алертов" {
		t.Fatalf("%q %q %v", title, body, ok)
	}
}

func TestMessage_groupOpened(t *testing.T) {
	single := model.Group{
		Severity: model.BucketCritical,
		Root:     model.Alert{Alertname: "DiskFull", Severity: model.BucketCritical},
		Members:  []model.Alert{{Alertname: "DiskFull"}},
	}
	title, body, ok := Message([]aggregate.Event{{Type: "group.opened", Cluster: "prod", Group: &single}})
	if !ok || title != "Штурвал" || body != "DiskFull · prod" {
		t.Fatalf("%q %q %v", title, body, ok)
	}

	members := make([]model.Alert, 5)
	members[0] = model.Alert{Alertname: "KubeNodeNotReady"}
	cascade := model.Group{
		Severity: model.BucketCritical,
		Root:     model.Alert{Alertname: "KubeNodeNotReady", Severity: model.BucketCritical},
		Members:  members,
	}
	_, body, ok = Message([]aggregate.Event{{Type: "group.opened", Cluster: "prod", Group: &cascade}})
	if !ok || body != "KubeNodeNotReady · prod · ещё 4" {
		t.Fatalf("%q %v", body, ok)
	}
}

func TestMessage_severalGroupsAndPromotion(t *testing.T) {
	a := model.Group{Root: model.Alert{Alertname: "A"}, Members: []model.Alert{{Alertname: "A"}}}
	b := model.Group{Root: model.Alert{Alertname: "B"}, Members: []model.Alert{{Alertname: "B"}}}
	_, body, ok := Message([]aggregate.Event{
		{Type: "group.opened", Cluster: "a", Group: &a},
		{Type: "group.opened", Cluster: "b", Group: &b},
	})
	if !ok || body != "2 новых групп" {
		t.Fatalf("%q %v", body, ok)
	}
	promoted := model.Group{
		Severity: model.BucketCritical,
		Root:     model.Alert{Alertname: "KubeNodeNotReady", Severity: model.BucketCritical},
		Members:  []model.Alert{{Alertname: "KubeNodeNotReady"}},
	}
	_, body, ok = Message([]aggregate.Event{{Type: "group.promoted", Cluster: "prod", Group: &promoted}})
	if !ok || body != "KubeNodeNotReady · prod · critical" {
		t.Fatalf("%q %v", body, ok)
	}
}

func TestMessage_ignoresEmptyDiff(t *testing.T) {
	if _, _, ok := Message(nil); ok {
		t.Fatal("expected no message")
	}
}

func TestMessage_ignoresSilence(t *testing.T) {
	if _, _, ok := Message([]aggregate.Event{
		{Type: "alert.silenced", Cluster: "prod", Alert: &model.Alert{Alertname: "DiskFull"}},
		{Type: "alert.unsilenced", Cluster: "prod", Alert: &model.Alert{Alertname: "DiskFull"}},
	}); ok {
		t.Fatal("silence must not notify")
	}
}
