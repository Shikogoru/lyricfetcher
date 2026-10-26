package cache

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"lyricfetcher/internal/metadata"
	"lyricfetcher/internal/provider"
)

func TestCacheOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cacheFile := filepath.Join(tempDir, "test_cache.json")
	c, err := New(cacheFile)
	if err != nil {
		t.Fatalf("failed to initialize cache: %v", err)
	}

	dur := 180
	info := &metadata.SongInfo{
		Title:    "Test Song",
		Artist:   "Test Artist",
		Album:    "Test Album",
		Duration: &dur,
	}

	lyr := &provider.Lyric{
		Content:      "[00:01.00] Test Lyric",
		Synced:       true,
		Source:       "lrclib",
		Confidence:   "high",
		Instrumental: false,
	}

	// 1. Initial get should be false
	if _, ok := c.Get(info); ok {
		t.Errorf("expected cache miss")
	}

	// 2. Put and Get
	c.Put(info, lyr)
	got, ok := c.Get(info)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if got.Content != lyr.Content || got.Source != lyr.Source {
		t.Errorf("got lyric %+v, want %+v", got, lyr)
	}

	// 3. Save to disk
	if err := c.Save(); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// 4. Reload from disk
	c2, err := New(cacheFile)
	if err != nil {
		t.Fatalf("failed to reload cache: %v", err)
	}
	got2, ok := c2.Get(info)
	if !ok {
		t.Fatalf("expected cache hit after reload")
	}
	if got2.Content != lyr.Content {
		t.Errorf("reloaded content mismatch: got %s, want %s", got2.Content, lyr.Content)
	}

	// 5. Clear
	if err := c2.Clear(); err != nil {
		t.Fatalf("failed to clear cache: %v", err)
	}
	if c2.Len() != 0 {
		t.Errorf("expected empty cache, got %d", c2.Len())
	}
	if _, ok := c2.Get(info); ok {
		t.Errorf("expected cache miss after clear")
	}
}

func TestCacheConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_conc_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cacheFile := filepath.Join(tempDir, "conc_cache.json")
	c, err := New(cacheFile)
	if err != nil {
		t.Fatalf("failed to initialize cache: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			dur := 100 + idx
			info := &metadata.SongInfo{
				Title:    "Song",
				Artist:   "Artist",
				Duration: &dur,
			}
			lyr := &provider.Lyric{
				Content: "Lyrics",
				Source:  "lrclib",
			}
			c.Put(info, lyr)
			c.Get(info)
		}()
	}
	wg.Wait()

	if c.Len() != 20 {
		t.Errorf("expected 20 entries, got %d", c.Len())
	}
}
