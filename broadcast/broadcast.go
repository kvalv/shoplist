package broadcast

import (
	"log/slog"
	"sync"
)

type Broadcast[T any] struct {
	subs []*subscriber[T]
	mu   sync.Mutex
	log  *slog.Logger
}

func New[T any](log *slog.Logger) *Broadcast[T] {
	return &Broadcast[T]{log: log}
}

func (b *Broadcast[T]) Publish(t T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.subs {
		if s.closed {
			continue
		}
		s.Ch <- t
	}
}

func (b *Broadcast[T]) Subscribe() *subscriber[T] {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan T, 10)
	sub := &subscriber[T]{
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
