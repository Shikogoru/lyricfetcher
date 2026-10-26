<#
.SYNOPSIS
    Cross-platform build script for LyricFetcher on Windows PowerShell.
.PARAMETER Target
    Build target: build, build-cli, build-gui, build-all, test, clean, release (default: build).
.PARAMETER Version
    Version string to inject (default: 1.0.0).
#>
param(
    [string]$Target = "build",
    [string]$Version = "1.0.0"
)

$ErrorActionPreference = "Stop"

$Commit = "dev"
try {
    $gitCommit = git rev-parse --short HEAD 2>$null
    if ($LASTEXITCODE -eq 0 -and $gitCommit) {
        $Commit = $gitCommit.Trim()
    }
} catch {}

$Date = Get-Date -Format "yyyy-MM-dd"
$LdFlags = "-s -w -X main.version=$Version -X main.commit=$Commit -X main.date=$Date"

# Auto-detect local w64devkit if installed in tools
$localDevkit = "C:\Users\Shikogoru\.gemini\antigravity\scratch\tools\w64devkit\bin"
if (Test-Path $localDevkit) {
    if ($env:PATH -notlike "*w64devkit*") {
        $env:PATH = "$localDevkit;" + $env:PATH
    }
}

function Clean-Dirs {
    Write-Host "Cleaning bin/ and dist/..."
    Remove-Item -Recurse -Force "bin", "dist" -ErrorAction SilentlyContinue
}

function Run-Tests {
    Write-Host "Running tests..."
    go test ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Tests failed!"
    }
}

function Build-CLI {
    Write-Host "Building LyricFetcher CLI (Version: $Version, Commit: $Commit)..."
    New-Item -ItemType Directory -Force "bin" | Out-Null
    $out = "bin/lyricfetcher"
    if ($IsWindows -or $env:OS -like "*Windows*") {
        $out = "bin/lyricfetcher.exe"
    }
    go build -ldflags $LdFlags -o $out ./cmd/cli
    if ($LASTEXITCODE -ne 0) {
        Write-Error "CLI build failed!"
    }
    Write-Host "Built successfully -> $out"
}

function Build-GUI {
    Write-Host "Building LyricFetcher GUI (Version: $Version, Commit: $Commit)..."
    New-Item -ItemType Directory -Force "bin" | Out-Null
    $out = "bin/lyricfetcher-gui"
    if ($IsWindows -or $env:OS -like "*Windows*") {
        $out = "bin/lyricfetcher-gui.exe"
    }
    $prevCgo = $env:CGO_ENABLED
    $env:CGO_ENABLED = "1"
    try {
        $guiFlags = $LdFlags
        if ($IsWindows -or $env:OS -like "*Windows*") {
            $guiFlags = "$LdFlags -H=windowsgui"
        }
        go build -ldflags $guiFlags -o $out ./cmd/gui
        if ($LASTEXITCODE -ne 0) {
            Write-Error "GUI build failed!"
        }
        Write-Host "Built successfully -> $out"
    } finally {
        $env:CGO_ENABLED = $prevCgo
    }
}

function Build-Current {
    Build-CLI
    $gcc = Get-Command "gcc" -ErrorAction SilentlyContinue
    if ($gcc) {
        Build-GUI
    } else {
        Write-Host "GCC not found, skipping GUI build."
    }
}

function Build-All {
    Write-Host "Building CLI for all 6 target platforms..."
    New-Item -ItemType Directory -Force "dist" | Out-Null

    $targets = @(
        @{ OS = "windows"; Arch = "amd64"; Ext = ".exe" },
        @{ OS = "windows"; Arch = "arm64"; Ext = ".exe" },
        @{ OS = "linux";   Arch = "amd64"; Ext = "" },
        @{ OS = "linux";   Arch = "arm64"; Ext = "" },
        @{ OS = "darwin";  Arch = "amd64"; Ext = "" },
        @{ OS = "darwin";  Arch = "arm64"; Ext = "" }
    )

    foreach ($t in $targets) {
        $name = "lyricfetcher-$Version-$($t.OS)-$($t.Arch)$($t.Ext)"
        $outPath = "dist/$name"
        Write-Host "  -> Building $outPath (GOOS=$($t.OS) GOARCH=$($t.Arch))..."
        
        if ($t.OS -eq "windows" -and $t.Arch -eq "amd64" -and (Get-Command "gcc" -ErrorAction SilentlyContinue)) {
            $env:CGO_ENABLED = "1"
            go build -ldflags $LdFlags -o $outPath ./cmd/cli
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed building for $($t.OS)/$($t.Arch)"
            }
            # Also build standalone GUI executable
            $guiOutPath = "dist/lyricfetcher-gui-$Version-$($t.OS)-$($t.Arch)$($t.Ext)"
            Write-Host "  -> Building $guiOutPath (Standalone GUI)..."
            go build -ldflags "$LdFlags -H=windowsgui" -o $guiOutPath ./cmd/gui
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed building standalone GUI for $($t.OS)/$($t.Arch)"
            }
        } else {
            $env:CGO_ENABLED = "0"
            go build -ldflags $LdFlags -o $outPath ./cmd/cli
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed building for $($t.OS)/$($t.Arch)"
            }
        }

        # Optional UPX compression
        $upx = Get-Command "upx" -ErrorAction SilentlyContinue
        if ($upx -and ($t.OS -eq "windows" -or $t.OS -eq "linux")) {
            Write-Host "     [UPX] Compressing $outPath..."
            & upx --best --quiet $outPath 2>$null
        }
    }

    # Reset environment variables
    Remove-Item env:GOOS -ErrorAction SilentlyContinue
    Remove-Item env:GOARCH -ErrorAction SilentlyContinue
    Remove-Item env:CGO_ENABLED -ErrorAction SilentlyContinue

    Write-Host "All targets built successfully in dist/."
}

function Generate-Checksums {
    Write-Host "Generating SHA256 checksums in dist/checksums.txt..."
    $distFiles = Get-ChildItem -Path "dist" -File | Where-Object { $_.Name -ne "checksums.txt" }
    $lines = @()
    foreach ($file in $distFiles) {
        $hash = (Get-FileHash -Path $file.FullName -Algorithm SHA256).Hash.ToLower()
        $lines += "$hash  $($file.Name)"
    }
    $lines | Out-File -FilePath "dist/checksums.txt" -Encoding ascii
    Write-Host "Checksums saved to dist/checksums.txt"
}

switch ($Target.ToLower()) {
    "clean" {
        Clean-Dirs
    }
    "test" {
        Run-Tests
    }
    "build-cli" {
        Build-CLI
    }
    "build-gui" {
        Build-GUI
    }
    "build" {
        Build-Current
    }
    "build-all" {
        Clean-Dirs
        Build-All
    }
    "release" {
        Clean-Dirs
        Run-Tests
        Build-All
        Generate-Checksums
    }
    default {
        Write-Host "Unknown target: $Target. Usage: .\build.ps1 -Target [build|build-cli|build-gui|build-all|test|clean|release]"
        exit 1
    }
}
