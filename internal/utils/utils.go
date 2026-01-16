package utils

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/dhowden/tag"
)

func SafePointer[T any](p *T) T {
	var pp T
	if p != nil {
		return *p
	}
	return pp
}

func EmptyStringNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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

func GetTrackCover(path string) *string {
	metadata := GetTrackFileMetadata(path)
	if metadata == nil {
		return nil
	}

	if p := metadata.Picture(); p != nil {
		mime := p.MIMEType
		data := base64.StdEncoding.EncodeToString(p.Data)
		picture := fmt.Sprintf("data:%s;base64,%s", mime, data)
		return &picture
	}

	return nil
}
