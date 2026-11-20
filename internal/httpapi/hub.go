package httpapi

import (
	"sync"

	"github.com/vutratenko/shturval-alert-wall/internal/aggregate"
	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

type Hub struct {
	mu          sync.RWMutex
	snapshot    model.Snapshot
	subscribers map[chan aggregate.Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[chan aggregate.Event]struct{})}
}

func (h *Hub) Snapshot() model.Snapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.snapshot
}

func (h *Hub) Update(next model.Snapshot) []aggregate.Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	var events []aggregate.Event
	if !h.snapshot.GeneratedAt.IsZero() {
		events = aggregate.Diff(h.snapshot, next)
	}
	h.snapshot = next
	snapEv := aggregate.Event{Type: "snapshot", Snap: &next, At: next.GeneratedAt}
	h.broadcast(snapEv)
	for _, ev := range events {
		h.broadcast(ev)
	}
	return events
}

func (h *Hub) Subscribe(buf int) chan aggregate.Event {
	if buf <= 0 {
		buf = 16
	}
	ch := make(chan aggregate.Event, buf)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan aggregate.Event) {
	h.mu.Lock()
	delete(h.subscribers, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *Hub) broadcast(ev aggregate.Event) {
	for ch := range h.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}