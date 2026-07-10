package safefile

import (
	"fmt"
	"os"
	"path/filepath"
)

func ReadFile(path string) ([]byte, error) {
	cleanPath, err := CleanFilePath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(cleanPath) // #nosec G304 -- CleanFilePath normalizes and verifies a file path.
}

func CleanFilePath(path string) (string, error) {
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", cleanPath)
	}
	return cleanPath, nil
}
