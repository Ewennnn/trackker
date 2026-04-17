package service

import (
	"log/slog"
	"sync"
)

type Broadcaster[T any] struct {
	log     *slog.Logger
	mu      sync.RWMutex
	nextId  int
	clients map[int]chan T
}

func NewBroadcaster[T any](log *slog.Logger) *Broadcaster[T] {
	return &Broadcaster[T]{
		log:     log,
		clients: make(map[int]chan T),
	}
}

func (b *Broadcaster[T]) Subscribe(buffer int) (chan T, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	channel := make(chan T, buffer)
	id := b.nextId
	b.clients[id] = channel
	b.nextId++

	b.log.Info("Client subscribe", "id", id, "total", len(b.clients))
	return channel, func() {
		b.unsubscribe(id)
	}
}

func (b *Broadcaster[T]) unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.clients[id]; ok {
		close(ch)
		delete(b.clients, id)
		b.log.Info("Client unsubscribe", "id", id, "total", len(b.clients))
	}
}

func (b *Broadcaster[T]) Broadcast(data T) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.clients {
		ch <- data
	}
}
