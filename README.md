[English](README.md) | [Tiếng Việt](README.vi.md) |

# LyricFetcher

[![Build](https://github.com/Shikogoru/lyricfetcher/actions/workflows/ci.yml/badge.svg)](https://github.com/Shikogoru/lyricfetcher/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Shikogoru/lyricfetcher)](go.mod)

**LyricFetcher** is a fast, robust, cross-platform tool available as both a high-efficiency **CLI** and an intuitive **Graphical User Interface (GUI)**. It automatically downloads synchronized and plain lyrics (`.lrc`) from [LRCLIB](https://lrclib.net) and [NetEase Cloud Music](https://music.163.com) for your local audio files. It is designed to effortlessly prepare lyrics for self-hosted music streaming servers like Navidrome

---
## Features

- **Multi-Format Metadata Extraction**: Reads embedded tags from MP3 (ID3v2), FLAC (Vorbis Comment), and M4A/MP4 audio containers.
- **Automated Metadata Normalization**: Unicode NFC normalization, automated hyphen/dash splitting for combined "Title - Artist" strings, and trailing artist suffix cleanup.
- **Smart 4-Tier Fallback Chain**:
  - Tier 1: Track + Artist + Album + Duration (exact match)
  - Tier 2: Track + Artist
  - Tier 3: Search fallback query
  - Tier 4: NetEase Music secondary fallback
- **High-Throughput Worker Pool**: Configurable concurrency (`--workers 1-16`) throttled by built-in token-bucket rate limiters (LRCLIB 4 req/s, NetEase 1 req/s).
- **Persistent Local Caching & Resume**:
  - Saves found lyrics in SHA-256 hashed cache file at standard OS config directories.
  - Instantly skips existing `.lrc` files without overhead on repeated runs.
- **Native i18n Support**: Full interactive and CLI output in English (`en`), Vietnamese (`vi`), and Chinese (`zh`).
- **Realtime TTY Progress & Clean Logs**: Interactive progress bar in terminals (`\r`) and non-interleaved atomic blocks in redirection/CI modes.
---

## How It Works?

```text
  Local Audio File (mp3 / flac / m4a)
                 │
                 ▼
     [1. Extract Audio Metadata] ──> Title, Artist, Album, Duration
                 │
                 ▼
       [2. Check Local Cache] ──(Hit)──> Write .lrc file
                 │ 
                 ▼
       [3. Query LRCLIB API] ──(Found)──> Cache & Write .lrc
                 │ 
                 ▼
     [4. Query NetEase Fallback] ──(Found)──> Cache & Write .lrc
                 │ 
                 ▼
          Log 
```

---

## Installation

### Recommended: Pre-Built Binaries
Most users do **not** need to install Go or GCC. Simply download the ready-to-run executable for your operating system from the [GitHub Releases](https://github.com/Shikogoru/lyricfetcher/releases) 
---
## Navidrome & Media Server Compatibility

LyricFetcher generates standard `.lrc` sidecar files in the format `<track-name>.lrc` alongside your audio files.

> [!IMPORTANT]
> **Navidrome Full Scan**: Navidrome's periodic quick scan detects new audio tracks, but does **not** automatically detect new `.lrc` files added next to existing audio tracks. After running LyricFetcher, trigger a **Full Scan** in Navidrome (`Settings` -> `Activity` -> `Start Full Scan`) so that Navidrome indexes the new `.lrc` files.

Compatible clients include:
- **Navidrome Web UI**: Synced lyrics displayed in the player bar.
- **Feishin (Desktop)**: Interactive synced scrolling panel.
- **Symfonium (Android)**: Full synced lyrics playback with offset controls.

See [Navidrome Workflow Guide](examples/navidrome-workflow.md) for detailed setup.

---
⚠️ Disclaimer
**NetEase Cloud Music** is an unofficial, reverse-engineered API used as a fallback source. This feature is provided for **personal and educational use only**. We do not host any copyrighted lyrics. Please respect the Terms of Service of NetEase and support the official platforms. The developers of LyricFetcher are not responsible for any misuse or account bans resulting from the use of this feature!

⚠️ 免责声明
**网易云音乐** 是一个非官方的逆向工程 API，仅作为备用歌词源使用。此功能仅供 **个人和学习目的** 使用。我们不托管任何受版权保护的歌词。请遵守网易的服务条款并支持官方平台。LyricFetcher 的开发者不对因使用此功能而导致的任何滥用或账户封禁负责。
