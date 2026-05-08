package api

import (
	"fmt"
	"net/http"
)

type SsePacket struct {
	http.ResponseWriter
	Event string
	Data  string
}

func (p *SsePacket) Format() string {
	packet := ""
	if p.Event != "" {
		packet += fmt.Sprintf("event: %s\n", p.Event)
	}

	if p.Data != "" {
		packet += fmt.Sprintf("data: %s\n", p.Data)
	}
	packet += "\n"
	return packet
}

type Sse struct {
	http.ResponseWriter
}

func (w *Sse) SendEvent(event, data string) error {
	packet := &SsePacket{
		Event: event,
		Data:  data,
	}
	return w.sendAndFlushPacket(packet)
}

// sendAndFlushPacket Écrit et envoie le packet SSE
func (w *Sse) sendAndFlushPacket(p *SsePacket) error {
	_, err := w.Write([]byte(p.Format()))
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
	return err
}

func (w *Sse) Ping() {
	_, _ = w.ResponseWriter.Write([]byte(": ping\n\n"))
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
