package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"lyricfetcher/internal/cache"
	"lyricfetcher/internal/console"
	"lyricfetcher/internal/i18n"
	"lyricfetcher/internal/metadata"
	"lyricfetcher/internal/provider"
	"lyricfetcher/internal/writer"
)

type Options struct {
	Workers         int
	LrclibProvider  provider.Provider
	NeteaseProvider provider.Provider
	NoNetease       bool
	Strict          bool
	DryRun          bool
	OutputDir       string
	Overwrite       bool
	NoCache         bool
	Cache           *cache.Cache
	LogFile         string
	ForceNonTTY     bool

	// Optional callbacks for GUI / progress observation
	OnFileStart func(file string)
	OnProgress  func(file string, processed, total, found, notFound, skipped, errors int)
}

type Summary struct {
	Total        int
	Found        int
	Synced       int
	Plain        int
	Instrumental int
	FromCache    int
	NotFound     int
	Skipped      int
	Errors       int
	Duration     time.Duration
	LogFile      string
}

func (s *Summary) Print() {
	console.Println()
	console.Println(i18n.T("summary_title"))
	console.Println(i18n.Tf("summary_total", s.Total))
	console.Println(i18n.Tf("summary_found", s.Found, s.Synced, s.Plain, s.Instrumental))
	console.Println(i18n.Tf("summary_from_cache", s.FromCache))
	console.Println(i18n.Tf("summary_not_found", s.NotFound))
	console.Println(i18n.Tf("summary_skipped", s.Skipped))
	console.Println(i18n.Tf("summary_errors", s.Errors))
	console.Println(i18n.Tf("summary_duration", s.Duration.Seconds()))
	console.Println(i18n.Tf("summary_log", s.LogFile))
}

func Run(ctx context.Context, files []string, opts Options) *Summary {
	startTime := time.Now()

	numWorkers := opts.Workers
	if numWorkers <= 0 {
		numWorkers = 4
	}
	if numWorkers > 16 {
		numWorkers = 16
	}

	logPath := opts.LogFile
	if logPath == "" {
		logPath = "lyricfetcher.log"
	}
	logger, _ := NewFileLogger(logPath)
	if logger != nil {
		defer logger.Close()
	}

	provider.DebugLogger = func(format string, args ...any) {
		if logger != nil {
			logger.Log("DEBUG", "provider", format, args...)
		}
	}

	// Sắp xếp danh sách file theo thứ tự bảng chữ cái để deterministic
	sortedFiles := make([]string, len(files))
	copy(sortedFiles, files)
	sort.Strings(sortedFiles)

	isTTY := IsTTY() && !opts.ForceNonTTY
	progress := NewProgress(len(sortedFiles), isTTY)

	lrclibLimiter := NewLRCLIBLimiter(4)
	defer lrclibLimiter.Stop()
	neteaseLimiter := NewNetEaseLimiter()

	summary := &Summary{
		Total:   len(sortedFiles),
		LogFile: logPath,
	}

	var (
		mu             sync.Mutex
		newCacheCount  int64
		saveTickerDone = make(chan struct{})
	)

	// Background cache saving ticker (every 30s)
	if !opts.NoCache && opts.Cache != nil {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-saveTickerDone:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					_ = opts.Cache.Save()
				}
			}
		}()
	}

	jobs := make(chan string, len(sortedFiles))
	for _, f := range sortedFiles {
		jobs <- f
	}
	close(jobs)

	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		workerID := fmt.Sprintf("w%d", w)
		go func(workerTag string) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case file, ok := <-jobs:
					if !ok {
						return
					}
					processFile(ctx, file, workerTag, opts, summary, progress, logger, lrclibLimiter, neteaseLimiter, &mu, &newCacheCount)
				}
			}
		}(workerID)
	}

	wg.Wait()
	close(saveTickerDone)

	// Save cache once more at the end
	if !opts.NoCache && opts.Cache != nil {
		_ = opts.Cache.Save()
	}

	progress.Finish()
	summary.Duration = time.Since(startTime)
	return summary
}

func getDestPath(file, outputDir string) string {
	return writer.GetLRCPath(file, outputDir)
}

func processFile(
	ctx context.Context,
	file string,
	workerTag string,
	opts Options,
	summary *Summary,
	progress *Progress,
	logger *FileLogger,
	lrclibLimiter *LRCLIBLimiter,
	neteaseLimiter *NetEaseLimiter,
	mu *sync.Mutex,
	newCacheCount *int64,
) {
	var block strings.Builder
	flush := func() {
		progress.PrintFileBlock(block.String())
	}

	reportProgress := func() {
		if opts.OnProgress != nil {
			mu.Lock()
			processed := summary.Found + summary.NotFound + summary.Skipped + summary.Errors
			f := summary.Found
			nf := summary.NotFound
			sk := summary.Skipped
			errs := summary.Errors
			mu.Unlock()
			opts.OnProgress(file, processed, summary.Total, f, nf, sk, errs)
		}
	}

	if opts.OnFileStart != nil {
		opts.OnFileStart(file)
	}

	destPath := getDestPath(file, opts.OutputDir)

	// 1. Resume check: if .lrc already exists and !overwrite, skip immediately without reading metadata or calling API
	if !opts.Overwrite {
		if _, err := os.Stat(destPath); err == nil {
			logger.Log("INFO", workerTag, "%s: skipped (already had .lrc)", file)
			block.WriteString(fmt.Sprintf("%s %s\n  %s %s\n", i18n.T("file_header"), file, i18n.T("warn_prefix"), i18n.T("warn_exists")))
			flush()

			mu.Lock()
			summary.Skipped++
			mu.Unlock()
			progress.Update(0, 0, 0)
			reportProgress()
			return
		}
	}

	// 2. Read metadata
	rawInfo, err := metadata.Read(file)
	if err != nil {
		logger.Log("WARN", workerTag, "%s: failed to read metadata: %v", file, err)
		block.WriteString(fmt.Sprintf("%s %s\n  %s %s\n", i18n.T("file_header"), file, i18n.T("warn_prefix"), i18n.Tf("err_read_meta", err)))
		flush()

		mu.Lock()
		summary.Errors++
		mu.Unlock()
		progress.Update(0, 0, 1)
		reportProgress()
		return
	}

	norm := metadata.Normalize(rawInfo)

	// Fallback metadata từ filename
	fallbackApplied := false
	if strings.TrimSpace(norm.Title) == "" {
		norm.Title = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		fallbackApplied = true
	}
	if strings.TrimSpace(norm.Artist) == "" {
		for _, dash := range []string{" - ", " – ", " — ", "-", "–", "—"} {
			if idx := strings.LastIndex(norm.Title, dash); idx > 0 {
				before := strings.TrimSpace(norm.Title[:idx])
				after := strings.TrimSpace(norm.Title[idx+len(dash):])
				if before != "" && after != "" {
					norm.Title = before
					norm.Artist = after
					fallbackApplied = true
					break
				}
			}
		}
	}
	if fallbackApplied {
		logMsg := fmt.Sprintf("[DEBUG] Fallback metadata: title=\"%s\", artist=\"%s\"", norm.Title, norm.Artist)
		logger.Log("DEBUG", workerTag, "%s", logMsg)
		if metadata.DebugMode {
			console.Printf("%s\n", logMsg)
		}
		log.Printf("%s", logMsg)
	}

	info := norm
	if strings.TrimSpace(info.Title) == "" {
		logger.Log("WARN", workerTag, "%s: %s", file, i18n.T("warn_no_title"))
		block.WriteString(fmt.Sprintf("%s %s\n  %s %s\n", i18n.T("file_header"), file, i18n.T("warn_prefix"), i18n.T("warn_no_title")))
		flush()

		mu.Lock()
		summary.Errors++
		mu.Unlock()
		progress.Update(0, 0, 1)
		reportProgress()
		return
	}

	durStr := i18n.T("unknown")
	if info.Duration != nil {
		durStr = fmt.Sprintf("%ds", *info.Duration)
	}
	block.WriteString(fmt.Sprintf("%s %s\n  %s\n", i18n.T("file_header"), file, i18n.Tf("format_line", info.Title, info.Artist, info.Album, durStr)))

	// 3. Check Cache
	var lyric *provider.Lyric
	var fromCache bool
	if !opts.NoCache && opts.Cache != nil {
		if cachedLyr, ok := opts.Cache.Get(info); ok {
			lyric = cachedLyr
			fromCache = true
			logger.Log("INFO", workerTag, "%s: cache hit (source=%s, confidence=%s)", file, lyric.Source, lyric.Confidence)
		}
	}

	// 4. API Fallback Chain if not in cache
	var lrclibErr error
	var neteaseErr error

	if lyric == nil {
		// LRCLIB
		if opts.LrclibProvider != nil {
			if err := lrclibLimiter.Wait(ctx); err == nil {
				lyric, lrclibErr = opts.LrclibProvider.Fetch(info, opts.Strict)
				if lrclibErr != nil {
					logger.Log("WARN", workerTag, "%s: lrclib error: %v", file, lrclibErr)
				} else if lyric != nil {
					logger.Log("INFO", workerTag, "%s: lrclib hit (confidence=%s, synced=%v)", file, lyric.Confidence, lyric.Synced)
				}
			}
		}

		// NetEase Fallback
		if lyric == nil && !opts.NoNetease && opts.NeteaseProvider != nil {
			release, err := neteaseLimiter.Acquire(ctx)
			if err == nil {
				lyric, neteaseErr = opts.NeteaseProvider.Fetch(info, opts.Strict)
				release()
				if neteaseErr != nil {
					logger.Log("WARN", workerTag, "%s: netease error: %v", file, neteaseErr)
				} else if lyric != nil {
					logger.Log("INFO", workerTag, "%s: netease hit (confidence=%s, synced=%v)", file, lyric.Confidence, lyric.Synced)
				}
			}
		}
	}

	// 5. Handle results (not found vs found)
	if lyric == nil {
		hasNetErr := (lrclibErr != nil) || (neteaseErr != nil)
		providers := []string{"lrclib"}
		if !opts.NoNetease {
			providers = append(providers, "netease")
		}
		provStr := strings.Join(providers, ", ")

		if hasNetErr {
			var errParts []string
			if lrclibErr != nil {
				errParts = append(errParts, fmt.Sprintf("lrclib: %v", lrclibErr))
			}
			if neteaseErr != nil {
				errParts = append(errParts, fmt.Sprintf("netease: %v", neteaseErr))
			}
			netErrStr := strings.Join(errParts, "; ")

			logger.Log("WARN", workerTag, "%s: %s", file, i18n.Tf("warn_not_found_net_err", provStr, netErrStr))
			block.WriteString(fmt.Sprintf("  %s %s\n", i18n.T("warn_prefix"), i18n.Tf("warn_not_found_net_err", provStr, netErrStr)))
			flush()

			mu.Lock()
			summary.Errors++ // Network error counts as Error per Bug 3
			mu.Unlock()
			progress.Update(0, 0, 1)
			reportProgress()
		} else {
			logger.Log("INFO", workerTag, "%s: %s", file, i18n.Tf("warn_not_found", provStr))
			block.WriteString(fmt.Sprintf("  %s %s\n", i18n.T("warn_prefix"), i18n.Tf("warn_not_found", provStr)))
			flush()

			mu.Lock()
			summary.NotFound++ // Pure not found counts as NotFound
			mu.Unlock()
			progress.Update(0, 1, 0)
			reportProgress()
		}
		return
	}

	// If found and not from cache, save to cache
	if !fromCache && !opts.NoCache && opts.Cache != nil {
		opts.Cache.Put(info, lyric)
		count := atomic.AddInt64(newCacheCount, 1)
		if count%50 == 0 {
			_ = opts.Cache.Save()
		}
	}

	// Write file or dry run
	if opts.DryRun {
		logger.Log("DRY-RUN", workerTag, "Would write %s (source=%s, confidence=%s, synced=%v)", destPath, lyric.Source, lyric.Confidence, lyric.Synced)
		block.WriteString(fmt.Sprintf("  %s\n", i18n.Tf("info_dry_run", destPath, lyric.Source, lyric.Confidence, lyric.Synced)))
		flush()
	} else {
		if lyric.Instrumental {
			block.WriteString(fmt.Sprintf("  %s\n", i18n.T("info_instrumental")))
		}

		_, err := writer.WriteLRC(file, lyric, opts.OutputDir, opts.Overwrite)
		if err != nil {
			logger.Log("ERROR", workerTag, "%s: write error %s: %v", file, destPath, err)
			block.WriteString(fmt.Sprintf("  %s %v\n", i18n.T("warn_prefix"), err))
			flush()

			mu.Lock()
			summary.Errors++
			mu.Unlock()
			progress.Update(0, 0, 1)
			reportProgress()
			return
		}

		logger.Log("INFO", workerTag, "%s: wrote %s successfully", file, destPath)
		block.WriteString(fmt.Sprintf("  %s\n", i18n.Tf("info_ok", destPath, lyric.Source, lyric.Confidence)))
		flush()
	}

	mu.Lock()
	summary.Found++
	if fromCache {
		summary.FromCache++
	}
	if lyric.Instrumental {
		summary.Instrumental++
	} else if lyric.Synced {
		summary.Synced++
	} else {
		summary.Plain++
	}
	mu.Unlock()

	progress.Update(1, 0, 0)
	reportProgress()
}
