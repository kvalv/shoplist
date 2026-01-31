package broadcast

import (
	"log"
	"sync"
)

type broadcast struct {
	subs []*subscriber
	mu   sync.Mutex
}

func New() *broadcast {
	return &broadcast{}
}

func (b *broadcast) Publish() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.subs {
		if s.closed {
			continue
		}
		s.Ch <- struct{}{}
	}
}

func (b *broadcast) Subscribe() *subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan struct{}, 10)
	sub := &subscriber{
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

type subscriber struct {
	closed bool
	Ch     chan struct{}
	close  func()
}

func (s *subscriber) Close() {
	s.close()
}
