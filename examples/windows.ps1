<#
.SYNOPSIS
    LyricFetcher - Windows PowerShell Usage Examples
#>

# 1. Fetch lyrics for a single track
.\lyricfetcher.exe "D:\Music\Taylor Swift\Cruel Summer.mp3"

# 2. Batch scan entire Music folder with 4 workers and custom log file
.\lyricfetcher.exe --workers 4 --log-file "my_fetch.log" "D:\Music"

# 3. Save all .lrc files into a separate folder (e.g. for Symfonium/Navidrome)
.\lyricfetcher.exe --output-dir "D:\Lyrics" "D:\Music"

# 4. Dry-run in Vietnamese language with strict confidence matching
.\lyricfetcher.exe --lang vi --dry-run --strict "D:\Music"
