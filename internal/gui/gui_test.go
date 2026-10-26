//go:build cgo

package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSupportedAudio(t *testing.T) {
	cases := []struct {
		path     string
		expected bool
	}{
		{"song.mp3", true},
		{"track.FLAC", true},
		{"audio.m4a", true},
		{"music.wav", false},
		{"cover.jpg", false},
		{"lyrics.lrc", false},
		{"archive.zip", false},
	}

	for _, c := range cases {
		if got := isSupportedAudio(c.path); got != c.expected {
			t.Errorf("isSupportedAudio(%q) = %v; want %v", c.path, got, c.expected)
		}
	}
}

func TestScanAudioFiles(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	_ = os.WriteFile(filepath.Join(tmpDir, "1.mp3"), []byte("dummy"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("dummy"), 0644)
	_ = os.WriteFile(filepath.Join(subDir, "2.flac"), []byte("dummy"), 0644)
	_ = os.WriteFile(filepath.Join(subDir, "3.m4a"), []byte("dummy"), 0644)
	_ = os.WriteFile(filepath.Join(subDir, "cover.png"), []byte("dummy"), 0644)

	files, err := scanAudioFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanAudioFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 audio files, got %d: %v", len(files), files)
	}
}
