package service

import (
	"context"
	"log/slog"
	"sync"
	"trackker/internal/model"
)

type Event interface{}

type TrackEvent struct {
	Track model.Track
}

type ControlEvent struct {
	DisplayMode model.DisplayMode
}

type DisplayEvent interface {
	IsDisplayEvent()
}

type TrackChangeEvent struct {
	Track model.Track
}

func (TrackChangeEvent) IsDisplayEvent() {}

type DisplayModeChangeEvent struct {
	Mode model.DisplayMode
}

func (DisplayModeChangeEvent) IsDisplayEvent() {}

type Multiplexer struct {
	log *slog.Logger

	in                  <-chan Event
	displayBroadcaster  *Broadcaster[DisplayEvent]
	controlsBroadcaster *Broadcaster[DisplayEvent]

	mu                 sync.Mutex
	currentDisplayMode model.DisplayMode
	currentTrack       *model.Track
}

func NewMultiplexer(log *slog.Logger, eventBus <-chan Event) *Multiplexer {
	return &Multiplexer{
		log: log,

		in:                  eventBus,
		displayBroadcaster:  NewBroadcaster[DisplayEvent](log),
		controlsBroadcaster: NewBroadcaster[DisplayEvent](log),
	}
}

func (m *Multiplexer) Init(tracker *Tracker, controls *Controls) {
	m.currentTrack = tracker.GetCurrentTrack()
	m.currentDisplayMode = controls.displayMode
}

func (m *Multiplexer) GetCurrentTrack() *model.Track {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentTrack
}

func (m *Multiplexer) SubscribeToDisplay() (chan DisplayEvent, UnsubscribeFunc) {
	ch, unsubscribe := m.displayBroadcaster.Subscribe(1)

	m.mu.Lock()
	defer m.mu.Unlock()

	go func() {
		ch <- TrackChangeEvent{
			Track: *m.currentTrack,
		}
		ch <- DisplayModeChangeEvent{
			Mode: m.currentDisplayMode,
		}
	}()

	return ch, unsubscribe
}

func (m *Multiplexer) SubscribeToControls() (chan DisplayEvent, UnsubscribeFunc) {
	ch, unsubscribe := m.controlsBroadcaster.Subscribe(1)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentDisplayMode == model.DisplayModeLive && m.currentTrack != nil {
		ch <- TrackChangeEvent{
			Track: *m.currentTrack,
		}
	} else {
		ch <- DisplayModeChangeEvent{
			Mode: m.currentDisplayMode,
		}
	}

	return ch, unsubscribe
}

// Run écoute les informations provenant des services Tracker et Controls
// et les transfert dans les channels de traitement du Multiplexer
func (m *Multiplexer) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-m.in:
				if !ok {
					return
				}
				switch evt.(type) {
				case TrackEvent:
					m.processTrackEvent(evt.(TrackEvent))
				case ControlEvent:
					m.processControlEvent(evt.(ControlEvent))
				}
			}
		}
	}()
}

func (m *Multiplexer) processTrackEvent(evt TrackEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentTrack = &evt.Track
	m.controlsBroadcaster.Broadcast(TrackChangeEvent{Track: *m.currentTrack})
	if m.currentDisplayMode == model.DisplayModeLive {
		m.displayBroadcaster.Broadcast(TrackChangeEvent{Track: *m.currentTrack})
	}
}

func (m *Multiplexer) processControlEvent(evt ControlEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentDisplayMode == evt.DisplayMode {
		return
	}
	m.currentDisplayMode = evt.DisplayMode

	m.controlsBroadcaster.Broadcast(DisplayModeChangeEvent{Mode: m.currentDisplayMode})
	m.displayBroadcaster.Broadcast(DisplayModeChangeEvent{Mode: m.currentDisplayMode})
	if evt.DisplayMode == model.DisplayModeLive {
		m.displayBroadcaster.Broadcast(TrackChangeEvent{Track: *m.currentTrack})
	}
}
