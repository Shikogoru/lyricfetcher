package metadata

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestDecodeMetadataString(t *testing.T) {
	// Helper to simulate dhowden/tag's decodeISO8859
	simulateTagISO8859 := func(b []byte) string {
		runes := make([]rune, len(b))
		for i, x := range b {
			runes[i] = rune(x)
		}
		return string(runes)
	}

	// 1. Empty string
	if got := decodeMetadataString(""); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}

	// 2. ASCII string
	if got := decodeMetadataString("Taylor Swift - 1989"); got != "Taylor Swift - 1989" {
		t.Errorf("expected ASCII string unchanged, got %q", got)
	}

	// 3. Genuine UTF-8 string (must not be double-decoded)
	utf8Chinese := "周杰伦 - 晴天"
	if got := decodeMetadataString(utf8Chinese); got != utf8Chinese {
		t.Errorf("expected UTF-8 Chinese string unchanged, got %q", got)
	}

	utf8Japanese := "宇多田ヒカル - First Love"
	if got := decodeMetadataString(utf8Japanese); got != utf8Japanese {
		t.Errorf("expected UTF-8 Japanese string unchanged, got %q", got)
	}

	utf8Vietnamese := "Đen Vâu - Trốn Tìm"
	if got := decodeMetadataString(utf8Vietnamese); got != utf8Vietnamese {
		t.Errorf("expected UTF-8 Vietnamese string unchanged, got %q", got)
	}

	// 4. Misdecoded GBK from tag (ISO-8859-1 runes)
	gbkEncoder := simplifiedchinese.GBK.NewEncoder()
	gbkArtistBytes, err := gbkEncoder.Bytes([]byte("周杰伦"))
	if err != nil {
		t.Fatal(err)
	}
	misdecodedGBK := simulateTagISO8859(gbkArtistBytes)
	if got := decodeMetadataString(misdecodedGBK); got != "周杰伦" {
		t.Errorf("expected decoded GBK '周杰伦', got %q", got)
	}

	gbkTitleBytes, err := gbkEncoder.Bytes([]byte("晴天"))
	if err != nil {
		t.Fatal(err)
	}
	misdecodedTitle := simulateTagISO8859(gbkTitleBytes)
	if got := decodeMetadataString(misdecodedTitle); got != "晴天" {
		t.Errorf("expected decoded GBK '晴天', got %q", got)
	}

	// 5. Raw GBK bytes string
	rawGBK := string(gbkArtistBytes)
	if got := decodeMetadataString(rawGBK); got != "周杰伦" {
		t.Errorf("expected decoded raw GBK '周杰伦', got %q", got)
	}

	// 6. Misdecoded Shift-JIS with Japanese Kana
	sjisEncoder := japanese.ShiftJIS.NewEncoder()
	sjisBytes, err := sjisEncoder.Bytes([]byte("サクラ"))
	if err == nil {
		misdecodedSJIS := simulateTagISO8859(sjisBytes)
		if got := decodeMetadataString(misdecodedSJIS); got != "サクラ" {
			t.Errorf("expected decoded Shift-JIS 'サクラ', got %q", got)
		}
	}

	// 7. Genuine Latin-1 accented string (must remain unchanged)
	latin1 := "Café"
	if got := decodeMetadataString(latin1); got != "Café" {
		t.Errorf("expected 'Café', got %q", got)
	}
}

func TestReadGBKMP3File(t *testing.T) {
	path := filepath.Join("..", "..", "test_gbk.mp3")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = "test_gbk.mp3"
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("test_gbk.mp3 not found, skipping file read test")
	}

	info, err := Read(path)
	if err != nil {
		t.Fatalf("Read(test_gbk.mp3) failed: %v", err)
	}

	if info.Artist != "周杰伦" {
		t.Errorf("expected Artist '周杰伦', got %q", info.Artist)
	}
	if info.Title != "晴天" {
		t.Errorf("expected Title '晴天', got %q", info.Title)
	}
	if info.Album != "叶惠美" {
		t.Errorf("expected Album '叶惠美', got %q", info.Album)
	}
}

func TestReadUTF8AudioFiles(t *testing.T) {
	// Verify existing sample files still decode cleanly without regression
	audioFiles := []string{
		filepath.Join("..", "..", "test_audio", "1.mp3"),
		filepath.Join("..", "..", "test_audio", "2.flac"),
		filepath.Join("..", "..", "test_audio", "3.m4a"),
	}

	for _, p := range audioFiles {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		info, err := Read(p)
		if err != nil {
			t.Errorf("Read(%s) failed: %v", p, err)
			continue
		}
		if info.Title == "" {
			t.Errorf("Read(%s) returned empty title", p)
		}
	}
}
