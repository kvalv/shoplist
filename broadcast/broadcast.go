package broadcast

import (
	"log"
	"sync"
)

type broadcast[T any] struct {
	subs []*subscriber[T]
	mu   sync.Mutex
}

func New[T any]() *broadcast[T] {
	return &broadcast[T]{}
}

func (b *broadcast[T]) Publish(t T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.subs {
		if s.closed {
			continue
		}
		s.Ch <- t
	}
}

func (b *broadcast[T]) Subscribe() *subscriber[T] {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan T, 10)
	sub := &subscriber[T]{
		Ch: ch,
	}

	sub.close = sync.OnceFunc(func() {
		log.Printf("[subscriber] Closing")
		b.mu.Lock()
		defer b.mu.Unlock()
		close(ch)
		sub.closed = true
		log.Printf("[subscriber] Closed")
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
