//go:build cgo

package gui

import (
	_ "embed"
	"image/color"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"lyricfetcher/internal/i18n"
)

//go:embed fonts/NotoSans-Regular.ttf
var internalNotoSansRegular []byte

//go:embed fonts/NotoSansCJKsc-Regular.otf
var internalNotoSansCJK []byte

var (
	regularFontResource fyne.Resource
	cjkFontResource     fyne.Resource
)

func initEmbeddedFonts() {
	if len(internalNotoSansRegular) > 0 {
		regularFontResource = fyne.NewStaticResource("NotoSans-Regular.ttf", internalNotoSansRegular)
	}
	if len(internalNotoSansCJK) > 0 {
		cjkFontResource = fyne.NewStaticResource("NotoSansCJKsc-Regular.otf", internalNotoSansCJK)
	}

	// Write fonts to local fonts/ directory and temp directory so FYNE_FONT works
	_ = os.MkdirAll("fonts", 0755)
	ttfPath := filepath.Join("fonts", "NotoSans-Regular.ttf")
	otfPath := filepath.Join("fonts", "NotoSansCJKsc-Regular.otf")

	if _, err := os.Stat(ttfPath); err != nil && len(internalNotoSansRegular) > 0 {
		_ = os.WriteFile(ttfPath, internalNotoSansRegular, 0644)
	}
	if _, err := os.Stat(otfPath); err != nil && len(internalNotoSansCJK) > 0 {
		_ = os.WriteFile(otfPath, internalNotoSansCJK, 0644)
	}

	tmpDir := filepath.Join(os.TempDir(), "lyricfetcher_fonts")
	_ = os.MkdirAll(tmpDir, 0755)
	tmpTTF := filepath.Join(tmpDir, "NotoSans-Regular.ttf")
	tmpOTF := filepath.Join(tmpDir, "NotoSansCJKsc-Regular.otf")
	if _, err := os.Stat(tmpTTF); err != nil && len(internalNotoSansRegular) > 0 {
		_ = os.WriteFile(tmpTTF, internalNotoSansRegular, 0644)
	}
	if _, err := os.Stat(tmpOTF); err != nil && len(internalNotoSansCJK) > 0 {
		_ = os.WriteFile(tmpOTF, internalNotoSansCJK, 0644)
	}

	// Set environment variables as requested
	_ = os.Setenv("FYNE_FONT", "fonts/NotoSans-Regular.ttf")
	_ = os.Setenv("FYNE_FONT_MONOSPACE", "fonts/NotoSansCJKsc-Regular.otf")
}

type embeddedTheme struct{}

var _ fyne.Theme = (*embeddedTheme)(nil)

func (embeddedTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Monospace || i18n.CurrentLang() == "zh" {
		if cjkFontResource != nil {
			return cjkFontResource
		}
	}
	if regularFontResource != nil {
		return regularFontResource
	}
	return theme.DefaultTheme().Font(style)
}

func (embeddedTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}

func (embeddedTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (embeddedTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}

// SetEmbeddedFonts allows external packages to supply embedded font bytes.
func SetEmbeddedFonts(regular, cjk []byte) {
	if len(regular) > 0 {
		internalNotoSansRegular = regular
		regularFontResource = fyne.NewStaticResource("NotoSans-Regular.ttf", regular)
	}
	if len(cjk) > 0 {
		internalNotoSansCJK = cjk
		cjkFontResource = fyne.NewStaticResource("NotoSansCJKsc-Regular.otf", cjk)
	}
}
