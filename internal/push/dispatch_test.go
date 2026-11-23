package push

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vutratenko/shturval-alert-wall/internal/aggregate"
	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

func TestDispatcher_dropsGoneSubscription(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(Subscription{Endpoint: "https://push.example/gone", Keys: Keys{P256dh: "p", Auth: "a"}}); err != nil {
		t.Fatal(err)
	}
	var got []byte
	d := &Dispatcher{Store: s, Send: func(_ context.Context, _ Subscription, payload []byte) error {
		got = payload
		return ErrGone
	}}
	d.Notify(context.Background(), []aggregate.Event{{
		Type: "alert.firing", Cluster: "prod", Alert: &model.Alert{Alertname: "DiskFull"},
	}})
	if len(s.List()) != 0 {
		t.Fatal("gone subscription was kept")
	}
	if !strings.Contains(string(got), "/wall") || !strings.Contains(string(got), "DiskFull") {
		t.Fatalf("payload %s", got)
	}
}

func TestDispatcher_skipsWhenNothingFired(t *testing.T) {
	s, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	d := &Dispatcher{Store: s, Send: func(context.Context, Subscription, []byte) error {
		return errors.New("should not send")
	}}
	d.Notify(context.Background(), []aggregate.Event{{Type: "alert.resolved"}})
}
