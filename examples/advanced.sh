#!/usr/bin/env bash
# LyricFetcher - Advanced Usage Example (Linux/macOS)

# 1. High throughput: 8 concurrent workers with overwrite enabled
lyricfetcher --workers 8 --overwrite "/home/user/Music"

# 2. Strict matching: require duration match within +/- 2s (high confidence)
lyricfetcher --strict "/home/user/Music"

# 3. Save all .lrc files into a centralized lyric directory (useful for media servers)
lyricfetcher --output-dir "/home/user/Lyrics" "/home/user/Music"

# 4. Use Vietnamese UI with NetEase fallback disabled (LRCLIB only)
lyricfetcher --lang vi --no-netease "/home/user/Music"

# 5. Clear persistent cache before executing batch
lyricfetcher --clear-cache "/home/user/Music"
