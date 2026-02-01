package events

import (
	"log/slog"
	"sync"
)

// A simple bus for publishing and subscribing to events of type T
type Bus struct {
	subs []*subscriber[Event]
	mu   sync.Mutex
	log  *slog.Logger
}

// Creates a new event bus
func NewBus(log *slog.Logger) *Bus {
	return &Bus{log: log}
}

func (b *Bus) Publish(t Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.subs {
		if s.closed {
			continue
		}
		s.Ch <- t
	}
}

func (b *Bus) Subscribe() *subscriber[Event] {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 10)
	sub := &subscriber[Event]{
		Ch: ch,
	}

	sub.close = sync.OnceFunc(func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		close(ch)
		sub.closed = true
		b.log.Info("subscriber closed")
	})

	b.subs = append(b.subs, sub)
	return sub
}

type subscriber[T any] struct {
	closed bool
	Ch     chan T
	close  func()
}

func (s *subscriber[T]) Close() {
	s.close()
}
