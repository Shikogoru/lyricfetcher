//go:build cgo

package main

import (
	_ "embed"
	"os"
	"path/filepath"

	"lyricfetcher/internal/gui"
)

//go:embed fonts/NotoSans-Regular.ttf
var notoSansRegular []byte

//go:embed fonts/NotoSansCJKsc-Regular.otf
var notoSansCJK []byte

func init() {
	_ = os.MkdirAll("fonts", 0755)
	ttfPath := filepath.Join("fonts", "NotoSans-Regular.ttf")
	otfPath := filepath.Join("fonts", "NotoSansCJKsc-Regular.otf")

	if _, err := os.Stat(ttfPath); err != nil && len(notoSansRegular) > 0 {
		_ = os.WriteFile(ttfPath, notoSansRegular, 0644)
	}
	if _, err := os.Stat(otfPath); err != nil && len(notoSansCJK) > 0 {
		_ = os.WriteFile(otfPath, notoSansCJK, 0644)
	}

	tmpDir := filepath.Join(os.TempDir(), "lyricfetcher_fonts")
	_ = os.MkdirAll(tmpDir, 0755)
	tmpTTF := filepath.Join(tmpDir, "NotoSans-Regular.ttf")
	tmpOTF := filepath.Join(tmpDir, "NotoSansCJKsc-Regular.otf")
	if _, err := os.Stat(tmpTTF); err != nil && len(notoSansRegular) > 0 {
		_ = os.WriteFile(tmpTTF, notoSansRegular, 0644)
	}
	if _, err := os.Stat(tmpOTF); err != nil && len(notoSansCJK) > 0 {
		_ = os.WriteFile(tmpOTF, notoSansCJK, 0644)
	}

	// Set font environment variables for Fyne
	os.Setenv("FYNE_FONT", "fonts/NotoSans-Regular.ttf")
	os.Setenv("FYNE_FONT_MONOSPACE", "fonts/NotoSansCJKsc-Regular.otf")

	gui.SetEmbeddedFonts(notoSansRegular, notoSansCJK)
}

func main() {
	gui.Run()
}
