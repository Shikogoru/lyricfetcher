package worker

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"lyricfetcher/internal/cache"
	"lyricfetcher/internal/i18n"
	"lyricfetcher/internal/metadata"
	"lyricfetcher/internal/provider"
)

const mockAudioB64 = "SUQzAwAAAAAIAFRJVDIAAAAVAAAB//5EAHIAaQB2AGUAIABCAHkAAABUUEUxAAAADwAAAf/+VAByAGEAaQBuAAAAVEFMQgAAAB8AAAH//kMAYQBsAGkAZgBvAHIAbgBpAGEAIAAzADcAAABUTEVOAAAAEQAAAf/+MQA5ADYAMAAwADAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/+5BkAAACQhzPVQxgAAAADSCgAAEZbVlvuWvAEAAANIMAAAAAAgAAAE5ABCru7gYAAInp1C/idROu7uHAAAQhfmif6IgGBuD5/4DB/u/3f8EAQOFwfD+7lwfB8Hw+CAIAgCADB8HwfggCAYpyWI4dS6fS7fbmzAQBAA6ZN9A2aIcPrLygOihigNngu24HPqNAGQ5WPQAwNgDFAwDwGCdnWBhYgaeB"

func getMockTestAudio(t *testing.T) []byte {
	srcAudio := filepath.Join("..", "..", "test_audio", "case2_driveby.mp3")
	if data, err := os.ReadFile(srcAudio); err == nil {
		return data
	}
	data, err := base64.StdEncoding.DecodeString(mockAudioB64)
	if err != nil {
		t.Fatalf("failed to decode mock audio: %v", err)
	}
	return data
}

type mockProvider struct {
	name       string
	fetchCount int64
	lyricFunc  func(info *metadata.SongInfo) (*provider.Lyric, error)
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Fetch(info *metadata.SongInfo, strict bool) (*provider.Lyric, error) {
	atomic.AddInt64(&m.fetchCount, 1)
	if m.lyricFunc != nil {
		return m.lyricFunc(info)
	}
	return &provider.Lyric{
		Content:    "[00:01.00] Mock Lyric",
		Synced:     true,
		Source:     m.name,
		Confidence: "high",
	}, nil
}

func TestWorkerPoolAndCache(t *testing.T) {
	_ = i18n.Init("en")

	tempDir, err := os.MkdirTemp("", "worker_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	var files []string
	for i := 1; i <= 6; i++ {
		p := filepath.Join(tempDir, fmt.Sprintf("song_%02d.mp3", i))
		if err := os.WriteFile(p, []byte("ID3dummy"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		files = append(files, p)
	}

	cacheFile := filepath.Join(tempDir, "cache.json")
	c, err := cache.New(cacheFile)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	p1 := &mockProvider{
		name: "lrclib",
	}

	opts := Options{
		Workers:        2,
		LrclibProvider: p1,
		NoNetease:      true,
		Cache:          c,
		OutputDir:      tempDir,
		DryRun:         true,
		ForceNonTTY:    true,
		LogFile:        filepath.Join(tempDir, "test.log"),
	}

	ctx := context.Background()
	summary := Run(ctx, files, opts)

	if summary.Total != 6 {
		t.Errorf("expected total 6, got %d", summary.Total)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewLRCLIBLimiter(10)
	defer limiter.Stop()

	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 80*time.Millisecond {
		t.Errorf("rate limiter did not throttle: elapsed %v", elapsed)
	}
}

func TestNetEaseLimiter(t *testing.T) {
	limiter := NewNetEaseLimiter()
	limiter.minInterval = 50 * time.Millisecond

	ctx := context.Background()
	start := time.Now()

	rel1, err := limiter.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire 1 failed: %v", err)
	}
	rel1()

	rel2, err := limiter.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire 2 failed: %v", err)
	}
	rel2()

	elapsed := time.Since(start)
	if elapsed < 40*time.Millisecond {
		t.Errorf("netease limiter did not wait: elapsed %v", elapsed)
	}
}

// BUG 1 TEST: Giả lập 1 file not found, chạy non-TTY, đảm bảo WARN chỉ xuất hiện 1 lần trong output
func TestBug1_NonTTYNoDuplicateWarn(t *testing.T) {
	_ = i18n.Init("en")

	tempDir, err := os.MkdirTemp("", "bug1_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dstAudio := filepath.Join(tempDir, "song_notfound.mp3")
	data := getMockTestAudio(t)
	if err := os.WriteFile(dstAudio, data, 0644); err != nil {
		t.Fatalf("failed to copy test audio: %v", err)
	}

	p := &mockProvider{
		name: "lrclib",
		lyricFunc: func(info *metadata.SongInfo) (*provider.Lyric, error) {
			return nil, nil // not found, no network error
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := Options{
		Workers:        2,
		LrclibProvider: p,
		NoNetease:      true,
		NoCache:        true,
		OutputDir:      tempDir,
		ForceNonTTY:    true,
		LogFile:        filepath.Join(tempDir, "test.log"),
	}

	summary := Run(context.Background(), []string{dstAudio}, opts)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	warnCount := strings.Count(output, "[WARN]")
	if warnCount != 1 {
		t.Errorf("expected [WARN] to appear exactly 1 time, but found %d times in output:\n%s", warnCount, output)
	}
	if summary.NotFound != 1 {
		t.Errorf("expected summary NotFound=1, got %d", summary.NotFound)
	}
}

// BUG 2 TEST: 4 worker cùng chạy 10 file non-TTY, verify mỗi block [FILE]...[OK]/[WARN] là liên tục, không bị chen
func TestBug2_NonTTYNoInterleaving(t *testing.T) {
	_ = i18n.Init("en")

	tempDir, err := os.MkdirTemp("", "bug2_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	data := getMockTestAudio(t)

	var files []string
	for i := 1; i <= 10; i++ {
		p := filepath.Join(tempDir, fmt.Sprintf("song_%02d.mp3", i))
		if err := os.WriteFile(p, data, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		files = append(files, p)
	}

	p := &mockProvider{
		name: "lrclib",
		lyricFunc: func(info *metadata.SongInfo) (*provider.Lyric, error) {
			time.Sleep(10 * time.Millisecond) // simulate concurrent work across workers
			return &provider.Lyric{
				Content:    "[00:01.00] Interleave test",
				Synced:     true,
				Source:     "lrclib",
				Confidence: "high",
			}, nil
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	opts := Options{
		Workers:        4,
		LrclibProvider: p,
		NoNetease:      true,
		NoCache:        true,
		OutputDir:      tempDir,
		ForceNonTTY:    true,
		LogFile:        filepath.Join(tempDir, "test.log"),
	}

	_ = Run(context.Background(), files, opts)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Every block starts with [FILE] and ends with [OK] or [WARN].
	// Split by [FILE] and verify each block contains exactly one result indicator without interleaving.
	blocks := strings.Split(output, "[FILE]")
	for i, b := range blocks {
		if i == 0 {
			continue // preamble before first [FILE]
		}
		hasOK := strings.Contains(b, "[OK]")
		hasWarn := strings.Contains(b, "[WARN]")
		if !hasOK && !hasWarn {
			t.Errorf("block %d missing result status ([OK] or [WARN]):\n%s", i, b)
		}
		if hasOK && hasWarn {
			t.Errorf("block %d contains both [OK] and [WARN]:\n%s", i, b)
		}
	}
	if len(blocks)-1 != 10 {
		t.Errorf("expected 10 blocks, got %d", len(blocks)-1)
	}
}

// BUG 3 TEST: 1 file not found + 1 file corrupt + 1 file no_tags -> Summary: NotFound=1, Errors=2
func TestBug3_SummaryClassification(t *testing.T) {
	_ = i18n.Init("en")

	tempDir, err := os.MkdirTemp("", "bug3_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	file1 := filepath.Join(tempDir, "valid.mp3")
	if err := os.WriteFile(file1, getMockTestAudio(t), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	file2 := filepath.Join(tempDir, "corrupt.mp3")
	if err := os.WriteFile(file2, []byte("corrupt mp3 data"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	file3 := filepath.Join(tempDir, "no_tags.mp3")
	if err := os.WriteFile(file3, make([]byte, 1024), 0644); err != nil {
		t.Fatalf("failed to write file3: %v", err)
	}

	p := &mockProvider{
		name: "lrclib",
		lyricFunc: func(info *metadata.SongInfo) (*provider.Lyric, error) {
			return nil, nil // Not found without network error
		},
	}

	opts := Options{
		Workers:        2,
		LrclibProvider: p,
		NoNetease:      true,
		NoCache:        true,
		OutputDir:      tempDir,
		ForceNonTTY:    true,
		LogFile:        filepath.Join(tempDir, "test.log"),
	}

	summary := Run(context.Background(), []string{file1, file2, file3}, opts)

	if summary.Total != 3 {
		t.Errorf("expected Total=3, got %d", summary.Total)
	}
	if summary.NotFound != 1 {
		t.Errorf("expected NotFound=1, got %d", summary.NotFound)
	}
	if summary.Errors != 2 {
		t.Errorf("expected Errors=2, got %d", summary.Errors)
	}
	if summary.Found != 0 {
		t.Errorf("expected Found=0, got %d", summary.Found)
	}
	if summary.Skipped != 0 {
		t.Errorf("expected Skipped=0, got %d", summary.Skipped)
	}
}

// BUG 4 TEST: 10 file audio + 10 file .lrc rỗng, chạy --workers 4, verify tổng thời gian < 1s và tất cả file skip
func TestBug4_ResumeSpeed(t *testing.T) {
	_ = i18n.Init("en")

	tempDir, err := os.MkdirTemp("", "bug4_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	var files []string
	for i := 1; i <= 10; i++ {
		audioPath := filepath.Join(tempDir, fmt.Sprintf("track_%02d.mp3", i))
		lrcPath := filepath.Join(tempDir, fmt.Sprintf("track_%02d.lrc", i))

		if err := os.WriteFile(audioPath, []byte("fake_audio"), 0644); err != nil {
			t.Fatalf("failed to write audio file: %v", err)
		}
		if err := os.WriteFile(lrcPath, []byte("[00:00.00] already exists"), 0644); err != nil {
			t.Fatalf("failed to write lrc file: %v", err)
		}
		files = append(files, audioPath)
	}

	opts := Options{
		Workers:     4,
		Overwrite:   false,
		NoCache:     true,
		ForceNonTTY: true,
		LogFile:     filepath.Join(tempDir, "test.log"),
	}

	start := time.Now()
	summary := Run(context.Background(), files, opts)
	elapsed := time.Since(start)

	if summary.Skipped != 10 {
		t.Errorf("expected Skipped=10, got %d", summary.Skipped)
	}
	if summary.Total != 10 {
		t.Errorf("expected Total=10, got %d", summary.Total)
	}
	if summary.Found != 0 || summary.NotFound != 0 || summary.Errors != 0 {
		t.Errorf("unexpected counts in summary: %+v", summary)
	}
	if elapsed >= 1*time.Second {
		t.Errorf("resume too slow: took %v, expected < 1s", elapsed)
	}
}

// BỔ SUNG TEST: Plain lyrics: mock provider trả Lyric{Synced: false, Content: "plain text"}
func TestPlainLyricsSummaryCount(t *testing.T) {
	_ = i18n.Init("en")

	tempDir, err := os.MkdirTemp("", "plain_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dstAudio := filepath.Join(tempDir, "song_plain.mp3")
	data := getMockTestAudio(t)
	if err := os.WriteFile(dstAudio, data, 0644); err != nil {
		t.Fatalf("failed to copy test audio: %v", err)
	}

	p := &mockProvider{
		name: "lrclib",
		lyricFunc: func(info *metadata.SongInfo) (*provider.Lyric, error) {
			return &provider.Lyric{
				Content:      "Line 1\nLine 2\nLine 3",
				Synced:       false,
				Source:       "lrclib",
				Confidence:   "high",
				Instrumental: false,
			}, nil
		},
	}

	opts := Options{
		Workers:        1,
		LrclibProvider: p,
		NoNetease:      true,
		NoCache:        true,
		OutputDir:      tempDir,
		ForceNonTTY:    true,
		LogFile:        filepath.Join(tempDir, "test.log"),
	}

	summary := Run(context.Background(), []string{dstAudio}, opts)

	if summary.Found != 1 {
		t.Errorf("expected Found=1, got %d", summary.Found)
	}
	if summary.Plain != 1 {
		t.Errorf("expected Plain=1, got %d", summary.Plain)
	}
	if summary.Synced != 0 {
		t.Errorf("expected Synced=0, got %d", summary.Synced)
	}
	if summary.Instrumental != 0 {
		t.Errorf("expected Instrumental=0, got %d", summary.Instrumental)
	}
}

func TestFallbackMetadataFromFilename(t *testing.T) {
	_ = i18n.Init("en")

	tempDir, err := os.MkdirTemp("", "fallback_meta_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dstAudio := filepath.Join(tempDir, "Custom Artist - Custom Title.mp3")
	data := getMockTestAudio(t)
	if err := os.WriteFile(dstAudio, data, 0644); err != nil {
		t.Fatalf("failed to copy test audio: %v", err)
	}

	p := &mockProvider{
		name: "lrclib",
		lyricFunc: func(info *metadata.SongInfo) (*provider.Lyric, error) {
			return &provider.Lyric{
				Content:      "[00:01.00] test",
				Synced:       true,
				Source:       "lrclib",
				Confidence:   "high",
				Instrumental: false,
			}, nil
		},
	}

	opts := Options{
		Workers:        1,
		LrclibProvider: p,
		NoNetease:      true,
		NoCache:        true,
		OutputDir:      tempDir,
		ForceNonTTY:    true,
		LogFile:        filepath.Join(tempDir, "test.log"),
	}

	summary := Run(context.Background(), []string{dstAudio}, opts)
	if summary.Found != 1 {
		t.Errorf("expected Found=1, got %d", summary.Found)
	}
}
