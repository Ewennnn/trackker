package utils

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"strings"
)

func SafeTrim(data *string) *string {
	if data == nil {
		return nil
	}
	v := strings.TrimSpace(*data)
	if v == "" {
		return nil
	}
	return &v
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return false
}

func SafeClose(file io.Closer, log *slog.Logger) {
	err := file.Close()
	if err != nil {
		log.Error("Failed to close file", err)
	}
}
