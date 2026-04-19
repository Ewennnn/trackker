package parser

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"trackker/internal/config"
	"trackker/internal/model"
)

const (
	virtualDJ = "virtualdj"
)

type Parser interface {

	// CheckState vérifie que le parser est dans un état valide pour démarrer le tracking
	CheckState() error

	// StartHistoryTracking démarre le suivi de l'historique des tracks,
	// en lisant les données à partir du reader et en envoyant les tracks trouvés dans le channel
	StartHistoryTracking(ctx context.Context, reader *bufio.Reader, ch chan *model.Track) error

	// WithHistoryTrackReader ouvre le fichier d'historique des tracks et fournit un bufio.Reader
	// à la fonction passée en argument pour lire les données.
	// La fonction doit gérer la logique de lecture et de traitement des données du reader.
	// Si une erreur survient lors de l'ouverture du fichier ou de la lecture, elle doit être retournée.
	WithHistoryTrackReader(fn func(reader *bufio.Reader) error) error

	// UpdateHistory lit le fichier d'historique actuellement chargé et met à jour les tracks non sauvegardées en base de données
	UpdateHistory(reader *bufio.Reader, track *model.Track) ([]*model.Track, error)
}

func GetParser(conf *config.Config, log *slog.Logger) (Parser, error) {
	var parser Parser

	switch conf.Tracker.History.Source {
	case virtualDJ:
		parser = &VirtualDJParser{
			log:  log,
			path: conf.Tracker.History.Path,
		}
	default:
		return nil, fmt.Errorf("unable find parser for %s source", conf.Tracker.History.Source)
	}

	return parser, nil
}
