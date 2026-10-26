package writer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lyricfetcher/internal/provider"
)

func TestWriteLRC(t *testing.T) {
	tempDir := t.TempDir()
	audioPath := filepath.Join(tempDir, "song.mp3")

	lyric := &provider.Lyric{
		Content:    "[00:01.00] Line 1\r\n[00:02.00] Line 2\r\n",
		Synced:     true,
		Source:     "lrclib",
		Confidence: "high",
	}

	// 1. Normal write
	outPath, err := WriteLRC(audioPath, lyric, "", false)
	if err != nil {
		t.Fatalf("WriteLRC failed: %v", err)
	}
	expectedPath := filepath.Join(tempDir, "song.lrc")
	if outPath != expectedPath {
		t.Errorf("expected outPath %s, got %s", expectedPath, outPath)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if strings.Contains(string(data), "\r\n") {
		t.Errorf("expected LF line endings, found CRLF")
	}

	// 2. ErrExists when overwrite = false
	_, errExists := WriteLRC(audioPath, lyric, "", false)
	if !errors.Is(errExists, ErrExists) {
		t.Errorf("expected ErrExists, got %v", errExists)
	}

	// 3. Overwrite = true
	_, errOverwrite := WriteLRC(audioPath, lyric, "", true)
	if errOverwrite != nil {
		t.Errorf("overwrite failed: %v", errOverwrite)
	}

	// 4. Custom outputDir
	customDir := filepath.Join(tempDir, "custom_out")
	outCustom, err := WriteLRC(audioPath, lyric, customDir, false)
	if err != nil {
		t.Fatalf("WriteLRC with outputDir failed: %v", err)
	}
	expectedCustom := filepath.Join(customDir, "song.lrc")
	if outCustom != expectedCustom {
		t.Errorf("expected %s, got %s", expectedCustom, outCustom)
	}

	// 5. Instrumental
	instLyric := &provider.Lyric{
		Instrumental: true,
		Source:       "lrclib",
		Confidence:   "high",
	}
	instPath, err := WriteLRC(filepath.Join(tempDir, "inst.mp3"), instLyric, "", false)
	if err != nil {
		t.Fatalf("WriteLRC instrumental failed: %v", err)
	}
	instData, _ := os.ReadFile(instPath)
	if string(instData) != "[instrumental]\n" {
		t.Errorf("expected '[instrumental]\\n', got %q", string(instData))
	}
}
