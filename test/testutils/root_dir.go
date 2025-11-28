package testutils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var rootDir string

func init() {
	rootDir = getRootDir(".")
}

func getRootDir(start string) string {
	absPath, err := filepath.Abs(start)
	if err != nil {
		return start
	}

	if strings.HasSuffix(absPath, "/goque") {
		return absPath
	}

	return getRootDir(filepath.Dir(absPath))
}

// GetPathFromRoot returns the absolute path to a file relative to the project root directory.
func GetPathFromRoot(file string) (string, error) {
	filePath := filepath.Join(rootDir, file)
	_, err := os.Stat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	return filepath.Abs(filePath)
}
