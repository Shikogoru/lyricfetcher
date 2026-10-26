# LyricFetcher — Build Instructions

This document provides step-by-step instructions for building **LyricFetcher** from source.

---

## Prerequisites

1. **Go**: Version `1.21` or higher is required.
   - Verify with: `go version`

2. **C Compiler (Required for GUI only)**:
   - **CLI**: The command-line tool is written in pure Go and requires **no CGO** or C compiler (`CGO_ENABLED=0`).
   - **GUI**: The Fyne GUI uses OpenGL and requires **CGO enabled** (`CGO_ENABLED=1`) along with a C compiler.

### Installing a C Compiler (Windows)
For Windows users building the GUI:
- **Recommended**: [w64devkit](https://github.com/skeeto/w64devkit/releases) (portable, zero-installer GCC distribution) or [MinGW-w64](https://www.mingw-w64.org/).
- Extract w64devkit and add its `bin` folder (containing `gcc.exe`) to your system `PATH`.
- Verify with: `gcc --version`

### Installing Dependencies (Linux)
On Debian / Ubuntu:
```bash
sudo apt update
sudo apt install -y gcc libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev
```

---

## Building the Command-Line Interface (CLI)

The CLI binary has no external C dependencies and compiles quickly across all platforms:

### Windows (PowerShell / CMD)
```powershell
# Set CGO off and compile CLI
$env:CGO_ENABLED = "0"
go build -ldflags "-s -w" -o bin/lyricfetcher.exe ./cmd/cli
```

### Linux / macOS
```bash
CGO_ENABLED=0 go build -ldflags "-s -w" -o bin/lyricfetcher ./cmd/cli
```

---

## Building the Graphical User Interface (GUI)

The GUI application embeds Google Noto Sans and Noto Sans CJK SC fonts so it works seamlessly out-of-the-box.

### Option A: Using the PowerShell Build Script (Windows)
The included `build.ps1` script automatically configures flags and handles output directory creation:
```powershell
.\build.ps1 -Target build
```

Or run directly without building to disk:
```powershell
.\build.ps1 -Target run
```

### Option B: Using Go CLI Directly (Windows)
```powershell
$env:CGO_ENABLED = "1"
go build -ldflags "-H=windowsgui -s -w" -o bin/lyricfetcher-gui.exe ./cmd/gui
```
> [!NOTE]
> The `-ldflags "-H=windowsgui"` flag prevents an empty command prompt window from launching in the background on Windows.

### Option C: Using Make (Linux / macOS)
```bash
make build-gui
```
Or directly with Go:
```bash
CGO_ENABLED=1 go build -ldflags "-s -w" -o bin/lyricfetcher-gui ./cmd/gui
```

---

## Running Automated Tests

To run the entire test suite:
```bash
go test -v ./...
```

To run tests with race condition detector (requires CGO):
```bash
go test -race ./...
```

---

## Code Quality & Verification

Before submitting code, ensure formatting and static analysis pass:

```bash
# Check code formatting (should produce no output)
gofmt -l .

# Run static analysis
go vet ./...
```
