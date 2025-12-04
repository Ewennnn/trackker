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

type SseWriter struct {
	http.ResponseWriter
}

func (w *SseWriter) WritePacket(p *SsePacket) error {
	_, err := w.Write([]byte(p.Format()))
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
	return err
}

func (w *SseWriter) Ping() (int, error) {
	return w.ResponseWriter.Write([]byte(": ping\n\n"))
}
