package service

import (
	"log/slog"
	"sync"
)

type UnsubscribeFunc func()

type Broadcaster[T any] struct {
	log     *slog.Logger
	mu      sync.RWMutex
	nextId  int
	clients map[int]chan T

	nextCountSubscriberID int
	countSubscribers      map[int]chan int
}

func NewBroadcaster[T any](log *slog.Logger) *Broadcaster[T] {
	return &Broadcaster[T]{
		log:              log,
		clients:          make(map[int]chan T),
		countSubscribers: make(map[int]chan int),
	}
}

func (b *Broadcaster[T]) Subscribe(buffer int) (chan T, UnsubscribeFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	channel := make(chan T, buffer)
	id := b.nextId
	b.clients[id] = channel
	b.nextId++
	b.notifyClientCountLocked()

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
		b.notifyClientCountLocked()
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

func (b *Broadcaster[T]) SubscribeClientCount(buffer int) (chan int, UnsubscribeFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan int, buffer)
	id := b.nextCountSubscriberID
	b.countSubscribers[id] = ch
	b.nextCountSubscriberID++

	ch <- len(b.clients)

	return ch, func() {
		b.unsubscribeClientCount(id)
	}
}

func (b *Broadcaster[T]) unsubscribeClientCount(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.countSubscribers[id]; ok {
		close(ch)
		delete(b.countSubscribers, id)
	}
}

func (b *Broadcaster[T]) notifyClientCountLocked() {
	count := len(b.clients)
	for _, ch := range b.countSubscribers {
		select {
		case ch <- count:
		default:
		}
	}
}

func (b *Broadcaster[T]) GetClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}
