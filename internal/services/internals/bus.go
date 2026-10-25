package internals

import (
	"sync"

	"taskmanager/models"
)

// Bus is an in-process fan-out for SSE subscribers.
type Bus struct {
	mu   sync.RWMutex
	subs map[chan models.EventEnvelope]struct{}
}

func NewBus() *Bus {
	return &Bus{subs: map[chan models.EventEnvelope]struct{}{}}
}

func (b *Bus) Subscribe() chan models.EventEnvelope {
	ch := make(chan models.EventEnvelope, 64)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Bus) Unsubscribe(ch chan models.EventEnvelope) {
	b.mu.Lock()
	if _, ok := b.subs[ch]; ok {
		delete(b.subs, ch)
		close(ch)
	}
	b.mu.Unlock()
}

func (b *Bus) Publish(event models.EventEnvelope) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- event:
		default:
		}
	}
}
