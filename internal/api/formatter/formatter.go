package formatter

import (
	"djtracker/internal/config"
	"djtracker/internal/model"
	"html/template"
	"log/slog"
)

type Formatter interface {
	Format(track *model.Track) (string, error)
}

func NewFormatter(cfg *config.Config, log *slog.Logger) (Formatter, error) {
	if cfg.Server.Format == "json" {
		return &JsonFormatter{}, nil
	}

	if cfg.Server.Format != "html" {
		log.Info("Unrecognized formatter value. Default html formatter will be used")
	}

	tmpl, err := template.ParseFiles("templates/current.html")
	if err != nil {
		return nil, err
	}

	return &HtmlFormatter{
		tmpl: tmpl,
	}, nil
}
