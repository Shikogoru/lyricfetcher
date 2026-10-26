package provider

import (
	"lyricfetcher/internal/console"
	"lyricfetcher/internal/metadata"
)

// Lyric represents fetched lyrics data.
type Lyric struct {
	Content      string // LRC content or plain text
	Synced       bool   // true if timestamps are present
	Source       string // "lrclib" or "netease"
	Confidence   string // "high" or "low"
	Instrumental bool   // true if instrumental track
}

// Provider defines the interface for lyrics retrieval services.
type Provider interface {
	Name() string
	// Fetch retrieves lyrics for the given song.
	// Returns (*Lyric, nil) on success,
	// (nil, nil) if not found or rejected by strict mode,
	// or (nil, err) on network / server failures.
	Fetch(info *metadata.SongInfo, strict bool) (*Lyric, error)
}

// DebugLogger is an optional callback to receive debug log messages.
var DebugLogger func(format string, args ...any)

func debugLog(format string, args ...any) {
	if metadata.DebugMode {
		console.Printf(format+"\n", args...)
	}
	if DebugLogger != nil {
		DebugLogger(format, args...)
	}
}
