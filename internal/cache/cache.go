package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"lyricfetcher/internal/metadata"
	"lyricfetcher/internal/provider"
)

type Entry struct {
	Lyric provider.Lyric `json:"lyric"`
}

type Cache struct {
	mu      sync.RWMutex
	path    string
	entries map[string]Entry
	dirty   bool
}

// DefaultCachePath returns the platform-specific default cache file path.
func DefaultCachePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "lyricfetcher", "cache.json"), nil
}

// New creates or loads a cache from the specified path (or default path if empty).
func New(customPath string) (*Cache, error) {
	path := customPath
	if path == "" {
		p, err := DefaultCachePath()
		if err != nil {
			return nil, err
		}
		path = p
	}

	c := &Cache{
		path:    path,
		entries: make(map[string]Entry),
	}
	_ = c.load()
	return c, nil
}

func (c *Cache) MakeKey(info *metadata.SongInfo) string {
	durStr := ""
	if info.Duration != nil {
		durStr = strconv.Itoa(*info.Duration)
	}
	raw := fmt.Sprintf("%s|%s|%s|%s",
		strings.ToLower(strings.TrimSpace(info.Title)),
		strings.ToLower(strings.TrimSpace(info.Artist)),
		strings.ToLower(strings.TrimSpace(info.Album)),
		durStr,
	)
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash)
}

func (c *Cache) Get(info *metadata.SongInfo) (*provider.Lyric, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.MakeKey(info)
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	lyr := entry.Lyric
	return &lyr, true
}

func (c *Cache) Put(info *metadata.SongInfo, lyric *provider.Lyric) {
	if lyric == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.MakeKey(info)
	c.entries[key] = Entry{Lyric: *lyric}
	c.dirty = true
}

func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]Entry)
	c.dirty = false
	if _, err := os.Stat(c.path); err == nil {
		return os.Remove(c.path)
	}
	return nil
}

func (c *Cache) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}
	var entries map[string]Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	c.entries = entries
	return nil
}

func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.dirty {
		return nil
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, "cache-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	_ = os.Remove(c.path)
	if err := os.Rename(tmpName, c.path); err != nil {
		os.Remove(tmpName)
		return err
	}

	c.dirty = false
	return nil
}

func (c *Cache) Path() string {
	return c.path
}

func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
