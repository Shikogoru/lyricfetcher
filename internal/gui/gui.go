//go:build cgo

package gui

import (
	"context"
	"errors"
	"fmt"
	"image/png"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"lyricfetcher/internal/cache"
	"lyricfetcher/internal/i18n"
	"lyricfetcher/internal/provider"
	"lyricfetcher/internal/worker"
)

var supportedAudioExts = map[string]bool{
	".mp3":  true,
	".flac": true,
	".m4a":  true,
}

func isSupportedAudio(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return supportedAudioExts[ext]
}

func scanAudioFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !d.IsDir() && isSupportedAudio(p) {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

func openInFileManager(path string) {
	cleanPath := filepath.Clean(path)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", cleanPath)
	case "darwin":
		cmd = exec.Command("open", cleanPath)
	default:
		cmd = exec.Command("xdg-open", cleanPath)
	}
	_ = cmd.Start()
}

func init() {
	initEmbeddedFonts()
}

// HasGUI returns true when GUI is compiled in with CGO.
func HasGUI() bool {
	return true
}

// Run launches the Fyne GUI.
func Run() {
	_ = i18n.Init("auto")

	a := app.NewWithID("com.lyricfetcher.gui")
	a.Settings().SetTheme(&embeddedTheme{})
	w := a.NewWindow("LyricFetcher")
	w.Resize(fyne.NewSize(640, 520))

	// Track processing state
	var (
		mu              sync.Mutex
		isRunning       bool
		cancelFunc      context.CancelFunc
		currentFound    int
		currentNotFound int
		currentSkipped  int
		currentErrors   int
	)

	// Language Selector
	langOptions := []string{"English", "Tiếng Việt", "中文"}
	langMap := map[string]string{
		"English":    "en",
		"Tiếng Việt": "vi",
		"中文":         "zh",
	}

	selectedLangText := "English"
	currentLang := i18n.CurrentLang()
	if currentLang == "vi" {
		selectedLangText = "Tiếng Việt"
	} else if currentLang == "zh" {
		selectedLangText = "中文"
	}

	langSelect := widget.NewSelect(langOptions, nil)
	langSelect.SetSelected(selectedLangText)

	// Top Bar: Title + Language Selector
	titleLabel := widget.NewLabelWithStyle("LyricFetcher", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	topBar := container.NewBorder(nil, nil, titleLabel, langSelect)

	// Folder Input
	folderLabel := widget.NewLabel(i18n.T("gui_folder_label"))
	folderEntry := widget.NewEntry()
	folderEntry.TextStyle = fyne.TextStyle{Monospace: true}
	folderEntry.SetPlaceHolder(filepath.FromSlash("C:/Music"))

	browseBtn := widget.NewButton(i18n.T("gui_browse_btn"), nil)

	// Options
	overwriteCheck := widget.NewCheck(i18n.T("gui_opt_overwrite"), nil)
	strictCheck := widget.NewCheck(i18n.T("gui_opt_strict"), nil)
	noNeteaseCheck := widget.NewCheck(i18n.T("gui_opt_no_netease"), nil)

	workersSlider := widget.NewSlider(1, 16)
	workersSlider.Step = 1
	workersSlider.SetValue(4)
	workersLabel := widget.NewLabel(i18n.Tf("gui_opt_workers", 4))
	workersSlider.OnChanged = func(val float64) {
		workersLabel.SetText(i18n.Tf("gui_opt_workers", int(val)))
	}

	optionsCard := container.NewVBox(
		overwriteCheck,
		strictCheck,
		noNeteaseCheck,
		container.NewBorder(nil, nil, workersLabel, nil, workersSlider),
	)

	optionsAccordion := widget.NewAccordion(
		widget.NewAccordionItem(i18n.T("gui_options_header"), optionsCard),
	)

	// Progress & Status
	progressBar := widget.NewProgressBar()
	progressBar.SetValue(0)

	statusLabel := widget.NewLabel(i18n.T("gui_status_ready"))
	statusLabel.TextStyle = fyne.TextStyle{Monospace: true}
	statusLabel.Truncation = fyne.TextTruncateEllipsis

	// Stats Labels
	foundLabel := widget.NewLabel(i18n.Tf("gui_stat_found", 0))
	notFoundLabel := widget.NewLabel(i18n.Tf("gui_stat_not_found", 0))
	skippedLabel := widget.NewLabel(i18n.Tf("gui_stat_skipped", 0))
	errorsLabel := widget.NewLabel(i18n.Tf("gui_stat_errors", 0))

	statsContainer := container.NewGridWithColumns(4, foundLabel, notFoundLabel, skippedLabel, errorsLabel)

	// Action Buttons
	startBtn := widget.NewButton(i18n.T("gui_btn_start"), nil)
	startBtn.Importance = widget.HighImportance

	stopBtn := widget.NewButton(i18n.T("gui_btn_stop"), nil)
	stopBtn.Importance = widget.DangerImportance
	stopBtn.Disable()

	openFolderBtn := widget.NewButton(i18n.T("gui_btn_open_folder"), nil)

	// Refresh UI texts on language switch
	updateTexts := func() {
		folderLabel.SetText(i18n.T("gui_folder_label"))
		browseBtn.SetText(i18n.T("gui_browse_btn"))
		overwriteCheck.SetText(i18n.T("gui_opt_overwrite"))
		strictCheck.SetText(i18n.T("gui_opt_strict"))
		noNeteaseCheck.SetText(i18n.T("gui_opt_no_netease"))
		workersLabel.SetText(i18n.Tf("gui_opt_workers", int(workersSlider.Value)))
		if len(optionsAccordion.Items) > 0 {
			optionsAccordion.Items[0].Title = i18n.T("gui_options_header")
			optionsAccordion.Refresh()
		}

		mu.Lock()
		running := isRunning
		f, nf, sk, errs := currentFound, currentNotFound, currentSkipped, currentErrors
		mu.Unlock()

		if !running {
			statusLabel.SetText(i18n.T("gui_status_ready"))
		}
		foundLabel.SetText(i18n.Tf("gui_stat_found", f))
		notFoundLabel.SetText(i18n.Tf("gui_stat_not_found", nf))
		skippedLabel.SetText(i18n.Tf("gui_stat_skipped", sk))
		errorsLabel.SetText(i18n.Tf("gui_stat_errors", errs))

		startBtn.SetText(i18n.T("gui_btn_start"))
		stopBtn.SetText(i18n.T("gui_btn_stop"))
		openFolderBtn.SetText(i18n.T("gui_btn_open_folder"))
	}

	langSelect.OnChanged = func(selected string) {
		if code, ok := langMap[selected]; ok {
			_ = i18n.Init(code)
			updateTexts()
		}
	}

	browseBtn.OnTapped = func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			folderEntry.SetText(uri.Path())
		}, w)
	}

	openFolderBtn.OnTapped = func() {
		targetDir := folderEntry.Text
		if targetDir == "" {
			dialog.ShowInformation(i18n.T("warn_prefix"), i18n.T("gui_err_no_dir"), w)
			return
		}
		if _, err := os.Stat(targetDir); err != nil {
			dialog.ShowError(err, w)
			return
		}
		openInFileManager(targetDir)
	}

	stopBtn.OnTapped = func() {
		mu.Lock()
		if isRunning && cancelFunc != nil {
			cancelFunc()
			statusLabel.SetText(i18n.T("gui_status_cancelled"))
			stopBtn.Disable()
		}
		mu.Unlock()
	}

	startBtn.OnTapped = func() {
		dir := strings.TrimSpace(folderEntry.Text)
		if dir == "" {
			dialog.ShowInformation(i18n.T("warn_prefix"), i18n.T("gui_err_no_dir"), w)
			return
		}

		stat, err := os.Stat(dir)
		if err != nil || !stat.IsDir() {
			dialog.ShowError(errors.New(i18n.T("gui_err_no_dir")), w)
			return
		}

		files, err := scanAudioFiles(dir)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		if len(files) == 0 {
			dialog.ShowInformation(i18n.T("warn_prefix"), i18n.T("gui_err_no_audio"), w)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())

		mu.Lock()
		isRunning = true
		cancelFunc = cancel
		currentFound = 0
		currentNotFound = 0
		currentSkipped = 0
		currentErrors = 0
		mu.Unlock()

		startBtn.Disable()
		stopBtn.Enable()
		browseBtn.Disable()
		folderEntry.Disable()
		openFolderBtn.Disable()

		progressBar.SetValue(0)
		statusLabel.SetText(i18n.Tf("gui_status_scanning", len(files)))
		foundLabel.SetText(i18n.Tf("gui_stat_found", 0))
		notFoundLabel.SetText(i18n.Tf("gui_stat_not_found", 0))
		skippedLabel.SetText(i18n.Tf("gui_stat_skipped", 0))
		errorsLabel.SetText(i18n.Tf("gui_stat_errors", 0))

		c, _ := cache.New("")

		opts := worker.Options{
			Workers:         int(workersSlider.Value),
			LrclibProvider:  provider.NewLRCLIBProvider(),
			NeteaseProvider: provider.NewNetEaseProvider(provider.NetEaseOptions{}),
			NoNetease:       noNeteaseCheck.Checked,
			Strict:          strictCheck.Checked,
			Overwrite:       overwriteCheck.Checked,
			Cache:           c,
			ForceNonTTY:     true,
			OnFileStart: func(f string) {
				fyne.Do(func() {
					statusLabel.SetText(i18n.Tf("gui_status_processing", filepath.Base(f)))
				})
			},
			OnProgress: func(f string, processed, total, found, notFound, skipped, errors int) {
				fyne.Do(func() {
					mu.Lock()
					currentFound = found
					currentNotFound = notFound
					currentSkipped = skipped
					currentErrors = errors
					mu.Unlock()

					if total > 0 {
						progressBar.SetValue(float64(processed) / float64(total))
					}
					foundLabel.SetText(i18n.Tf("gui_stat_found", found))
					notFoundLabel.SetText(i18n.Tf("gui_stat_not_found", notFound))
					skippedLabel.SetText(i18n.Tf("gui_stat_skipped", skipped))
					errorsLabel.SetText(i18n.Tf("gui_stat_errors", errors))
				})
			},
		}

		totalFiles := len(files)

		go func() {
			summary := worker.Run(ctx, files, opts)

			fyne.Do(func() {
				mu.Lock()
				isRunning = false
				cancelFunc = nil
				mu.Unlock()

				progressBar.SetValue(1.0)
				statusLabel.SetText(i18n.Tf("gui_status_completed", summary.Total, totalFiles))

				startBtn.Enable()
				stopBtn.Disable()
				browseBtn.Enable()
				folderEntry.Enable()
				openFolderBtn.Enable()

				// Show summary alert dialog
				msg := fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n%s\n%s",
					i18n.T("summary_title"),
					i18n.Tf("summary_total", summary.Total),
					i18n.Tf("summary_found", summary.Found, summary.Synced, summary.Plain, summary.Instrumental),
					i18n.Tf("summary_from_cache", summary.FromCache),
					i18n.Tf("summary_not_found", summary.NotFound),
					i18n.Tf("summary_skipped", summary.Skipped),
					i18n.Tf("summary_errors", summary.Errors),
				)
				dialog.ShowInformation(i18n.T("summary_title"), msg, w)

				if screenPath := os.Getenv("LYRICFETCHER_SCREENSHOT_PATH"); screenPath != "" {
					time.AfterFunc(600*time.Millisecond, func() {
						fyne.Do(func() {
							img := w.Canvas().Capture()
							if img != nil {
								_ = os.MkdirAll(filepath.Dir(screenPath), 0755)
								if f, err := os.Create(screenPath); err == nil {
									_ = png.Encode(f, img)
									_ = f.Close()
								}
							}
							if os.Getenv("LYRICFETCHER_AUTO_EXIT") == "1" {
								w.Close()
							}
						})
					})
				}
			})
		}()
	}

	// Layout construction
	folderRow := container.NewBorder(nil, nil, folderLabel, browseBtn, folderEntry)
	buttonsRow := container.NewGridWithColumns(3, startBtn, stopBtn, openFolderBtn)

	content := container.NewVBox(
		topBar,
		widget.NewSeparator(),
		folderRow,
		optionsAccordion,
		widget.NewSeparator(),
		progressBar,
		statusLabel,
		statsContainer,
		widget.NewSeparator(),
		buttonsRow,
	)

	// Pre-populate folder if provided via CLI or env var
	autoFolder := os.Getenv("LYRICFETCHER_AUTO_FOLDER")
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			if stat, err := os.Stat(arg); err == nil && stat.IsDir() {
				autoFolder = arg
				break
			}
		}
	}
	if autoFolder != "" {
		folderEntry.SetText(autoFolder)
		if os.Getenv("LYRICFETCHER_AUTO_START") == "1" {
			time.AfterFunc(300*time.Millisecond, func() {
				fyne.Do(func() {
					startBtn.OnTapped()
				})
			})
		}
	}

	w.SetContent(content)
	w.ShowAndRun()
}
