package formatter

import (
	"embed"
	"html/template"
	"log/slog"
	"trackker/internal/config"
	"trackker/internal/model"
)

//go:embed current.html
var trackHtmlTemplate embed.FS

// Formatter will be deleted in the future. Tracks data will be sent as json and the frontend will be responsible for formatting it as needed.
type Formatter interface {
	Format(track model.Track) (string, error)
}

func NewFormatter(cfg *config.Config, log *slog.Logger) (Formatter, error) {
	if cfg.Server.Format == "json" {
		return &JsonFormatter{}, nil
	}

	if cfg.Server.Format != "html" {
		log.Info("Unrecognized formatter value. Default html formatter will be used")
	}

	tmpl, err := template.ParseFS(trackHtmlTemplate, "current.html")
	if err != nil {
		return nil, err
	}

	return &HtmlFormatter{
		tmpl: tmpl,
	}, nil
}
