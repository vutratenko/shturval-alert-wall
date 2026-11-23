package push

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/aggregate"
)

var ErrGone = errors.New("push subscription gone")

type Sender func(ctx context.Context, sub Subscription, payload []byte) error

type Dispatcher struct {
	Store *Store
	Send  Sender
}

func (d *Dispatcher) Notify(ctx context.Context, events []aggregate.Event) {
	if d == nil || d.Store == nil || d.Send == nil {
		return
	}
	title, body, ok := Message(events)
	if !ok {
		return
	}
	payload, err := json.Marshal(struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		URL   string `json:"url"`
	}{Title: title, Body: body, URL: "/wall"})
	if err != nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	for _, sub := range d.Store.List() {
		err := d.Send(ctx, sub, payload)
		if errors.Is(err, ErrGone) {
			if delErr := d.Store.Delete(sub.Endpoint); delErr != nil {
				slog.Warn("push delete", "err", delErr)
			}
			continue
		}
		if err != nil {
			slog.Warn("push send", "err", err)
		}
	}
}
