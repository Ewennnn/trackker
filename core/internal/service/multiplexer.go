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

type DisplayModeEvent struct {
	DisplayMode model.DisplayMode
}

type DisplayServerStatus struct {
	Running bool
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

type DisplayServerStatusChangeEvent struct {
	Running bool
}

func (DisplayServerStatusChangeEvent) IsDisplayEvent() {}

type ConnectedClientEvent struct {
	Service string
	Count   int
}

func (ConnectedClientEvent) IsDisplayEvent() {}

type Multiplexer struct {
	log *slog.Logger

	in                  <-chan Event
	displayBroadcaster  *Broadcaster[DisplayEvent]
	controlsBroadcaster *Broadcaster[DisplayEvent]

	mu                         sync.Mutex
	currentDisplayMode         model.DisplayMode
	currentTrack               *model.Track
	currentDisplayServerStatus bool
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
	m.currentDisplayServerStatus = controls.GetDisplayServerStatus()
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

func (m *Multiplexer) SubscribeToControls() (chan DisplayEvent, UnsubscribeFunc) {
	ch, unsubscribe := m.controlsBroadcaster.Subscribe(1)

	m.mu.Lock()
	defer m.mu.Unlock()

	go func() {
		if m.currentTrack != nil {
			ch <- TrackChangeEvent{Track: *m.currentTrack}
		}
		ch <- DisplayModeChangeEvent{Mode: m.currentDisplayMode}
		ch <- DisplayServerStatusChangeEvent{Running: m.currentDisplayServerStatus}
		ch <- ConnectedClientEvent{Service: "display", Count: m.displayBroadcaster.GetClientCount()}
		// Request controls client count not needed, send by goroutine in listenConnectedClients
	}()

	return ch, unsubscribe
}

// Run lance les goroutines permettant au service Multiplexer d'être réactif aux évènements
// provenant des services Tracker et Controls.
func (m *Multiplexer) Run(ctx context.Context, wg *sync.WaitGroup) {
	go m.listenConnectedClients(ctx, wg)
	go m.listenIncomingEvents(ctx, wg)
}

// Run écoute les informations provenant des services Tracker et Controls
// et les transfert dans les channels de traitement du Multiplexer
func (m *Multiplexer) listenIncomingEvents(ctx context.Context, wg *sync.WaitGroup) {
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
				case DisplayModeEvent:
					m.processControlEvent(evt.(DisplayModeEvent))
				case DisplayServerStatus:
					m.processDisplayServerStatusEvent(evt.(DisplayServerStatus))
				}
			}
		}
	}()
}

// listenConnectedClients écoute les changements du nombre de clients connectés sur les services Tracker et Controls
func (m *Multiplexer) listenConnectedClients(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		displayClients, unsubscribeDisplayCount := m.displayBroadcaster.SubscribeClientCount(1)
		defer unsubscribeDisplayCount()
		controlsClients, unsubscribeControlsCount := m.controlsBroadcaster.SubscribeClientCount(1)
		defer unsubscribeControlsCount()
		for {
			select {
			case <-ctx.Done():
				return
			case count := <-displayClients:
				m.controlsBroadcaster.Broadcast(ConnectedClientEvent{Service: "display", Count: count})
			case count := <-controlsClients:
				m.controlsBroadcaster.Broadcast(ConnectedClientEvent{Service: "controls", Count: count})
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

func (m *Multiplexer) processControlEvent(evt DisplayModeEvent) {
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

func (m *Multiplexer) processDisplayServerStatusEvent(evt DisplayServerStatus) {
	m.currentDisplayServerStatus = evt.Running
	m.controlsBroadcaster.Broadcast(DisplayServerStatusChangeEvent{Running: m.currentDisplayServerStatus})
}
