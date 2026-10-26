package writer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lyricfetcher/internal/provider"
)

// ErrExists is returned when the destination .lrc file already exists and overwrite is false.
var ErrExists = errors.New("file already exists")

// GetLRCPath calculates the target .lrc file path for a given audio file and optional outputDir.
func GetLRCPath(audioPath string, outputDir string) string {
	ext := filepath.Ext(audioPath)
	base := strings.TrimSuffix(filepath.Base(audioPath), ext) + ".lrc"
	dir := filepath.Dir(audioPath)
	if strings.TrimSpace(outputDir) != "" {
		dir = outputDir
	}
	return filepath.Join(dir, base)
}

// WriteLRC safely writes lyric content to a .lrc file corresponding to audioPath.
// If overwrite is false and the file exists, ErrExists is returned.
// Safe writing uses a temporary file and atomic rename.
func WriteLRC(audioPath string, lyric *provider.Lyric, outputDir string, overwrite bool) (string, error) {
	if lyric == nil {
		return "", errors.New("cannot write nil lyric")
	}

	targetPath := GetLRCPath(audioPath, outputDir)
	targetDir := filepath.Dir(targetPath)

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return targetPath, fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	if _, err := os.Stat(targetPath); err == nil {
		if !overwrite {
			return targetPath, ErrExists
		}
	}

	content := lyric.Content
	if lyric.Instrumental {
		content = "[instrumental]\n"
	} else {
		// Normalize line endings to LF
		content = strings.ReplaceAll(content, "\r\n", "\n")
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
	}

	// Write atomically via temporary file
	tmpFile, err := os.CreateTemp(targetDir, ".lrc-tmp-*")
	if err != nil {
		return targetPath, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		// Clean up temp file if still present
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return targetPath, fmt.Errorf("failed to write lyric content: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return targetPath, fmt.Errorf("failed to close temp file: %w", err)
	}

	// On Windows, os.Rename fails if the target file already exists, even with overwrite
	if overwrite {
		_ = os.Remove(targetPath)
	}

	if err := os.Rename(tmpName, targetPath); err != nil {
		return targetPath, fmt.Errorf("failed to finalize lrc file: %w", err)
	}

	return targetPath, nil
}
