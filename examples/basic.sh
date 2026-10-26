#!/usr/bin/env bash
# LyricFetcher - Basic Usage Example (Linux/macOS)

# 1. Fetch lyrics for a single audio file (saves song.lrc in the same directory)
lyricfetcher "song.mp3"

# 2. Recursively scan an entire music folder using default 4 workers
lyricfetcher "/home/user/Music"

# 3. Simulate fetching without writing any .lrc files (dry-run)
lyricfetcher --dry-run "/home/user/Music"
