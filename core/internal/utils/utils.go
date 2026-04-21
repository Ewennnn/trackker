package utils

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/dhowden/tag"
)

func EmptyStringNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func StringPtrEqual(s1, s2 *string) bool {
	if s1 == nil && s2 == nil {
		return true
	}
	if s1 == nil || s2 == nil {
		return false
	}
	return *s1 == *s2
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

func SafeClose(file io.Closer) {
	err := file.Close()
	if err != nil {
		log.Panicln("Failed to close file", err)
	}
}

func GetTrackFileMetadata(path string) tag.Metadata {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer SafeClose(file)

	metadata, err := tag.ReadFrom(file)
	return metadata
}

func GetTrackCover(path string) *tag.Picture {
	metadata := GetTrackFileMetadata(path)
	if metadata == nil {
		return nil
	}

	return metadata.Picture()
}

func GetCoverURL(r *http.Request, trackID int64) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s/cover/%d", scheme, r.Host, trackID)
}
