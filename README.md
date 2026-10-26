[English](README.md) | [Tiếng Việt](README.vi.md) | [中文](README.zh.md)

# LyricFetcher

[![Build](https://github.com/Shikogoru/lyricfetcher/actions/workflows/ci.yml/badge.svg)](https://github.com/Shikogoru/lyricfetcher/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Shikogoru/lyricfetcher)](go.mod)

**LyricFetcher** is a fast, robust, cross-platform tool available as both a high-efficiency **CLI** and an intuitive **Graphical User Interface (GUI)**. It automatically downloads synchronized and plain lyrics (`.lrc`) from [LRCLIB](https://lrclib.net) and [NetEase Cloud Music](https://music.163.com) for your local audio files. It is designed to effortlessly prepare lyrics for self-hosted music streaming servers like **Navidrome**, **Feishin**, and **Symfonium**.

---

## Graphical User Interface (GUI)

LyricFetcher includes an official graphical interface built with [Fyne](https://fyne.io):

- **Folder Selection**: Intuitive directory picker and path input.
- **Collapsible Options**: Easily toggle overwrite, strict matching, NetEase fallback, and concurrent worker slider (1-16).
- **Real-Time Progress & Statistics**: Responsive progress bar, current file indicator, and live count badges for Found, Not Found, Skipped, and Errors.
- **Start / Stop Controls**: Safely start or interrupt batch processing anytime.
- **Instant Language Switching**: Toggle between English, Tiếng Việt, and 中文 seamlessly from the top-right menu.
- **One-Click Folder Access**: Open the target music library folder directly upon completion.

To launch the GUI:
```bash
./bin/lyricfetcher-gui
# On Windows:
.\bin\lyricfetcher-gui.exe
```

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

## Quick Start

```bash
# Fetch lyrics for a single track
lyricfetcher "01. Drive By.mp3"

# Scan an entire music collection with 4 concurrent workers
lyricfetcher "/path/to/Music"

# Dry run simulation in Vietnamese
lyricfetcher --lang vi --dry-run "/path/to/Music"
```

---

## How It Works

```text
  Local Audio File (mp3 / flac / m4a)
                 │
                 ▼
     [1. Extract Audio Metadata] ──> Title, Artist, Album, Duration
                 │
                 ▼
       [2. Check Local Cache] ──(Hit)──> Write .lrc file
                 │ (Miss)
                 ▼
       [3. Query LRCLIB API] ──(Found)──> Cache & Write .lrc
                 │ (Not Found)
                 ▼
     [4. Query NetEase Fallback] ──(Found)──> Cache & Write .lrc
                 │ (Not Found)
                 ▼
          Log [Not Found]
```

---

## Installation

### Recommended: Pre-Built Binaries
Most users do **not** need to install Go or GCC. Simply download the ready-to-run executable for your operating system from the [GitHub Releases](https://github.com/Shikogoru/lyricfetcher/releases) page:
- **GUI (Graphical Interface)**:
  - Windows: `lyricfetcher-gui-windows-amd64.exe` (includes embedded CJK Noto Sans fonts)
- **CLI (Command Line)**:
  - Windows: `lyricfetcher-windows-amd64.exe` or `windows-arm64.exe`
  - Linux: `lyricfetcher-linux-amd64` or `linux-arm64`
  - macOS: `lyricfetcher-darwin-amd64` or `darwin-arm64`

### Build from Source
If you prefer to compile from source:
- **CLI Only (Pure Go, no CGO or GCC required)**:
  ```bash
  git clone https://github.com/Shikogoru/lyricfetcher.git
  cd lyricfetcher
  go build -o bin/lyricfetcher ./cmd/cli
  ```
- **GUI Application (Requires CGO & GCC)**:
  Building the Fyne GUI from source requires `CGO_ENABLED=1` and a working C compiler (e.g. GCC via [w64devkit](https://github.com/skeeto/w64devkit) or MinGW-w64 on Windows; `gcc` / `clang` on Linux/macOS):
  ```bash
  # Windows PowerShell:
  .\build.ps1 -Target build

  # Linux / macOS:
  make build
  ```

---

## CLI Options

| Category | Flag | Default | Description |
| :--- | :--- | :--- | :--- |
| **General** | `--lang` | `auto` | Language: `en`, `vi`, `zh`, `auto` |
| | `--version` | `false` | Print version information and exit |
| | `--help` | `false` | Display help screen with examples |
| **Processing** | `--workers` | `4` | Number of concurrent workers (1 - 16) |
| | `--strict` | `false` | Require high-confidence metadata match (+/- 2s duration) |
| | `--no-netease` | `false` | Disable NetEase fallback provider |
| | `--netease-spoof-ip` | `false` | Enable NetEase IP spoofing header |
| | `--netease-spoof-ip-value` | `118.89.204.198` | IP value sent in NetEase `X-Real-IP` header |
| **Output** | `--output-dir` | `""` | Directory to save `.lrc` files (default: beside audio files) |
| | `--overwrite` | `false` | Overwrite existing `.lrc` files |
| | `--dry-run` | `false` | Simulate execution without modifying or creating files |
| | `--log-file` | `lyricfetcher.log` | Destination path for detailed ISO 8601 log file |
| **Cache** | `--no-cache` | `false` | Disable reading and writing persistent cache |
| | `--clear-cache` | `false` | Empty persistent cache before processing |

---

## Examples

### 1. Single Track Fetching
```bash
lyricfetcher "California 37/01. Drive By.mp3"
```

### 2. Fast Directory Batch with 8 Workers
```bash
lyricfetcher --workers 8 "/media/music"
```

### 3. Strict Match for High-Fidelity Libraries
Ensure that lyrics strictly match the duration of your audio file within ±2 seconds:
```bash
lyricfetcher --strict "/media/music"
```

### 4. Separate Output Directory
Save all lyrics into a centralized directory:
```bash
lyricfetcher --output-dir "/media/lyrics" "/media/music"
```

### 5. Overwrite and Clear Cache
Force re-downloading fresh lyrics across your entire library:
```bash
lyricfetcher --clear-cache --overwrite "/media/music"
```

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

