package parser

import (
	"bufio"
	"djtracker/internal/config"
	"djtracker/internal/model"
	"fmt"
	"log/slog"
)

const (
	virtualDJ = "virtualdj"
)

type Parser interface {
	CheckState() error
	StartHistoryTracking(reader *bufio.Reader, ch chan *model.Track) error
	WithHistoryTrackReader(fn func(reader *bufio.Reader) error) error
}

func GetParser(conf *config.Config, log *slog.Logger) (Parser, error) {
	var parser Parser

	switch conf.Tracker.History.Source {
	case virtualDJ:
		parser = &VirtualDJParser{
			log:     log,
			path:    conf.Tracker.History.Path,
			readAll: false,
		}
	default:
		return nil, fmt.Errorf("unable find parser for %s source", conf.Tracker.History.Source)
	}

	return parser, nil
}
