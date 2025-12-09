package service

import (
	"encoding/base64"
	"fmt"
	"github.com/dhowden/tag"
	"github.com/hcl/audioduration"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileTrackData struct {
	Path string
	Name string
	Ext  string
}

// MapExtType mappe l'extension du fichier vers un entier
// représentant la valeur attendu par la librairie audioduration.Duration()
func (f *FileTrackData) MapExtType() int {
	if f == nil {
		return -1
	}

	switch strings.ToLower(f.Ext) {
	case ".flac":
		return 0
	case ".mp3":
		return 2
	default:
		return -1
	}
}

// FindFile cherche le fichier name dans le dossier root
func FindFile(root, name string) (*FileTrackData, error) {
	var found *FileTrackData
	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(info.Name())
		trimName := strings.TrimSuffix(info.Name(), ext)

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(trimName), strings.ToLower(name)) {
			found = &FileTrackData{
				Path: path,
				Name: trimName,
				Ext:  ext,
			}
			return fs.SkipAll
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if found == nil || found.Name == "" {
		return nil, fmt.Errorf("file %s not found", name)
	}

	return found, nil
}

func (s *Service) findTrackDuration(file *os.File, extType int) (time.Duration, error) {
	duration, err := audioduration.Duration(file, extType)
	if err != nil {
		return -1, err
	}

	return time.Duration(duration * float64(time.Second)), nil
}

func (s *Service) findTrackCover(file *os.File) (string, error) {
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return "", err
	}

	if picture := metadata.Picture(); picture != nil {
		mime := picture.MIMEType
		data := base64.StdEncoding.EncodeToString(picture.Data)
		return fmt.Sprintf("data:%s;base64,%s", mime, data), nil
	}

	return "", fmt.Errorf("no picture found for this track")
}
