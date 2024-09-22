package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

func LegacyStoreFilename(path string, mediaID int64) string {
	h := sha256.Sum224([]byte(path))
	fn := hex.EncodeToString(h[:])
	extIdx := strings.LastIndex(path, ".")
	if extIdx > 0 {
		fn += path[extIdx:]
	}

	hexID := strconv.FormatInt(mediaID, 16)
	l := len(hexID)
	if l < 4 {
		hexID = strings.Repeat("0", l-4) + hexID
	}

	return filepath.Join(
		hexID[0:2],
		hexID[2:4],
		fn,
	)
}

func StoreFilename(path string) (string, error) {
	b := make([]byte, 20)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random name: %w", err)
	}
	fn := hex.EncodeToString(b)

	extIdx := strings.LastIndex(path, ".")
	if extIdx > 0 {
		fn += strings.ToLower(path[extIdx:])
	}

	return filepath.Join(
		fn[0:2],
		fn[2:4],
		fn,
	), nil
}

func GetFileType(name string) (string, error) {
	idx := strings.LastIndex(name, ".")
	if idx < 1 {
		return "", errors.New("extension not found in file name")
	}

	switch strings.ToLower(name[idx:]) {
	case ".jpg", ".jpeg", ".png", ".webm", ".heif", ".heic", ".heiff":
		return "photo", nil
	case ".gif":
		return "gif", nil
	case ".mp4", ".m4v", ".mov", ".avi", ".av1":
		return "video", nil
	case ".mp3", ".m4a", ".wav":
		return "audio", nil
	default:
		return "", fmt.Errorf("extension is not known: %s", name[idx:])
	}
}
