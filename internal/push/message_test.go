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
