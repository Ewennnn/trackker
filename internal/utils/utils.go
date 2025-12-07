package utils

import (
	"errors"
	"io/fs"
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
