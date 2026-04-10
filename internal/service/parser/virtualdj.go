package parser

import (
	"bufio"
	"context"
	"djtracker/internal/model"
	"djtracker/internal/utils"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var trackPrefix = "#EXTVDJ:"

type virtualDjTrack struct {
	Time         string  `xml:"time"`
	LastPlayTime int64   `xml:"lastplaytime"`
	FileSize     int64   `xml:"filesize"`
	Artist       string  `xml:"artist"`
	Title        string  `xml:"title"`
	Remix        string  `xml:"remix"`
	SongLength   float64 `xml:"songlength"`
	Path         string
}

func (t *virtualDjTrack) mapToTrack() *model.Track {
	return &model.Track{
		Artist:   utils.EmptyStringNil(t.Artist),
		Name:     t.Title,
		PlayAt:   time.Unix(t.LastPlayTime, 0),
		Path:     t.Path,
		Duration: time.Duration(t.SongLength * float64(time.Second)),
	}
}

type VirtualDJParser struct {
	log  *slog.Logger
	path string
}

func (p *VirtualDJParser) CheckState() error {
	if stats, err := os.Stat(p.path); err != nil || !stats.IsDir() {
		return fmt.Errorf("history directory path must be specified for VirtualDJ tracklist source (%s)", p.path)
	}
	return nil
}

// getHistoryTracksPath cherche le fichier d'historique de VirtualDJ (déclaré dans la configuration)
// Le fichier en cours d'utilisation est au format '.m3u', nommé à date actuelle au format 'YYYY-MM-DD'
// ou à date passée d'un jour avant 9h du jour courant.
func (p *VirtualDJParser) getHistoryTracksPath() (string, error) {
	files, err := os.ReadDir(p.path)
	if err != nil {
		return "", errors.New("failed to open directory " + p.path)
	}

	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".m3u" || len(file.Name()) < 10 {
			continue
		}

		dateStr := file.Name()[:10]
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if fileDate.Equal(today) || fileDate.Equal(yesterday) && now.Hour() < 9 {
			return filepath.Join(p.path, file.Name()), nil
		}
	}

	return "", errors.New("no history file found")
}

func (p *VirtualDJParser) WithHistoryTrackReader(fn func(reader *bufio.Reader) error) error {
	path, err := p.getHistoryTracksPath()
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer utils.SafeClose(file)

	reader := bufio.NewReader(file)

	return fn(reader) // Appel de la logique
}

// StartHistoryTracking lit le fichier d'historique et convertit les informations dans un format normalisé au programme.
func (p *VirtualDJParser) StartHistoryTracking(ctx context.Context, reader *bufio.Reader, ch chan *model.Track) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			data, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					time.Sleep(200 * time.Millisecond)
					continue
				}
				p.log.Error("Error while reading file", err)
				return err
			}

			if !strings.HasPrefix(data, trackPrefix) {
				p.log.Error("No prefix detected")
				continue
			}

			path, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					p.log.Error("Reached end of file on second line")
					continue
				}
				p.log.Error("Error while reading file", err)
				return err
			}

			track, err := p.parseStringTrackData(data, path)
			if err != nil {
				p.log.Error("Error on parse track data", "err", err)
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch <- track:
			}
		}
	}
}

// UpdateHistory lit le fichier d'historique m3u et retourne les tracks non enregistrées.
// Si track est nil, retourne toutes les tracks du fichier.
// Sinon, retourne les tracks qui viennent après la track de référence passée en paramètre.
// Le fichier est lu ligne par ligne (format m3u : alternance de données XML et chemins de fichiers).
func (p *VirtualDJParser) UpdateHistory(reader *bufio.Reader, track *model.Track) ([]*model.Track, error) {
	var newTracks []*model.Track
	// Si track est nil, on retourne toutes les tracks du fichier
	foundTrack := track == nil

	for {
		// Lire la ligne de données (contient les infos XML)
		data, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Fin du fichier atteinte
				break
			}
			return nil, err
		}

		// Ignorer les lignes qui ne contiennent pas les données de track
		if !strings.HasPrefix(data, trackPrefix) {
			continue
		}

		// Lire la ligne de chemin du fichier
		path, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		// Parser la track depuis les deux lignes
		currentTrack, err := p.parseStringTrackData(data, path)
		if err != nil {
			continue
		}

		// Si on a trouvé la track de référence, commencer à collectionner les suivantes
		if foundTrack {
			newTracks = append(newTracks, currentTrack)
		} else if currentTrack.Equals(track) {
			// Marquer qu'on a trouvé la track de référence
			foundTrack = true
		}
	}

	return newTracks, nil
}

func (p *VirtualDJParser) parseStringTrackData(data, path string) (*model.Track, error) {
	reformattedData := sanitizeXML(data)
	trackData, err := unmarshalXML(reformattedData)
	if err != nil {
		return nil, err
	}

	trackData.Path = strings.TrimSpace(path)
	return trackData.mapToTrack(), nil
}

func sanitizeXML(data string) string {
	data = strings.TrimSpace(data)
	data = strings.Replace(data, trackPrefix, "", 1)
	data = strings.ReplaceAll(data, "&", "&amp;")
	return "<track>" + data + "</track>"
}

func unmarshalXML(data string) (*virtualDjTrack, error) {
	var t virtualDjTrack
	if err := xml.Unmarshal([]byte(data), &t); err != nil {
		return nil, err
	}
	return &t, nil
}
