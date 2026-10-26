package metadata

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dhowden/tag"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// ErrNoMetadata is returned when an audio file has no tags or all metadata fields are empty.
var ErrNoMetadata = errors.New("no metadata found")

// SongInfo contains extracted metadata for an audio file.
type SongInfo struct {
	Path     string
	Artist   string
	Title    string
	Album    string
	Duration *int // Duration in seconds; nil if unknown
}

// HasAnyMetadata returns true if at least one metadata field is present.
func (s *SongInfo) HasAnyMetadata() bool {
	return s.Artist != "" || s.Title != "" || s.Album != "" || s.Duration != nil
}

// Equal returns true if Artist, Title, Album, and Duration are identical.
func (s *SongInfo) Equal(other *SongInfo) bool {
	if s == nil || other == nil {
		return s == other
	}
	if s.Artist != other.Artist || s.Title != other.Title || s.Album != other.Album {
		return false
	}
	if (s.Duration == nil) != (other.Duration == nil) {
		return false
	}
	if s.Duration != nil && *s.Duration != *other.Duration {
		return false
	}
	return true
}

// Normalize cleans and standardizes metadata fields according to Phase 1.5 rules:
// a) Trims leading and trailing whitespace from Artist, Title, and Album.
// b) Normalizes Artist, Title, and Album to Unicode NFC.
// c) If Artist is empty and Title contains "-", "–", or "—", splits on the LAST dash
//
//	taking the preceding part as Title and succeeding part as Artist.
//
// d) If Artist already exists and Title ends with " - {Artist}" (after trim), removes that suffix.
// e) Returns a new SongInfo without modifying the input, preserving Path and Duration.
func Normalize(info *SongInfo) *SongInfo {
	if info == nil {
		return nil
	}

	artist := norm.NFC.String(strings.TrimSpace(info.Artist))
	title := norm.NFC.String(strings.TrimSpace(info.Title))
	album := norm.NFC.String(strings.TrimSpace(info.Album))

	// Rule c: If Artist is empty and Title contains dash, split at the last dash
	if artist == "" {
		lastIdx := -1
		dashLen := 0
		for _, dash := range []string{"-", "–", "—"} {
			idx := strings.LastIndex(title, dash)
			if idx > lastIdx {
				lastIdx = idx
				dashLen = len(dash)
			}
		}

		if lastIdx != -1 {
			before := norm.NFC.String(strings.TrimSpace(title[:lastIdx]))
			after := norm.NFC.String(strings.TrimSpace(title[lastIdx+dashLen:]))
			if before != "" && after != "" {
				title = before
				artist = after
			}
		}
	} else {
		// Rule d: If Artist already exists and Title ends with " - {Artist}" or starts with "{Artist} - ", strip it
		for _, dash := range []string{" - ", " – ", " — "} {
			suffix := dash + artist
			if strings.HasSuffix(title, suffix) {
				title = norm.NFC.String(strings.TrimSpace(strings.TrimSuffix(title, suffix)))
				break
			}
			prefix := artist + dash
			if strings.HasPrefix(title, prefix) {
				title = norm.NFC.String(strings.TrimSpace(strings.TrimPrefix(title, prefix)))
				break
			}
		}
	}

	var durCopy *int
	if info.Duration != nil {
		d := *info.Duration
		durCopy = &d
	}

	return &SongInfo{
		Path:     info.Path,
		Artist:   artist,
		Title:    title,
		Album:    album,
		Duration: durCopy,
	}
}

// DebugMode controls whether metadata reading prints raw bytes as hex.
var DebugMode bool

func printHexDebug(field, raw string) {
	if !DebugMode {
		return
	}
	b := []byte(raw)
	limit := 50
	if len(b) < limit {
		limit = len(b)
	}
	fmt.Printf("[DEBUG] %s raw bytes (%d): %x\n", field, len(b), b[:limit])
}

// decodeMetadataString attempts to decode raw tag metadata strings that may have
// been improperly encoded or misdecoded by the tag library as ISO-8859-1.
// It tries in order: UTF-8 -> GBK -> Shift-JIS -> Big5.
func decodeMetadataString(raw string) string {
	if raw == "" {
		return ""
	}

	// 0. If raw is not valid UTF-8, decode directly from GBK
	if !utf8.ValidString(raw) {
		if s, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), raw); err == nil && utf8.ValidString(s) {
			return s
		}
	}

	var b []byte
	if !utf8.ValidString(raw) {
		b = []byte(raw)
	} else {
		hasHighLatin1 := false
		for _, r := range raw {
			if r > 0xFF {
				// Contains runes > 255 (already decoded UTF-8 / UTF-16, e.g. Chinese, Japanese, Vietnamese)
				return raw
			}
			if r >= 0x80 {
				hasHighLatin1 = true
			}
		}
		if !hasHighLatin1 {
			// Pure ASCII (< 128), no decoding needed
			return raw
		}

		// Reconstruct original bytes from dhowden/tag's decodeISO8859
		runes := []rune(raw)
		b = make([]byte, len(runes))
		for i, r := range runes {
			b[i] = byte(r)
		}
	}

	// 1. Try UTF-8 on the raw bytes (e.g. UTF-8 tagged as ISO-8859-1)
	if utf8.Valid(b) {
		s := string(b)
		if containsCJK(s) {
			return s
		}
	}

	// 2. Check if Shift-JIS produces genuine Japanese Kana (Hiragana / Katakana)
	if s, err := tryDecode(japanese.ShiftJIS.NewDecoder(), b); err == nil && containsKana(s) {
		return s
	}

	// 3. Try GBK (Simplified Chinese)
	if s, err := tryDecode(simplifiedchinese.GBK.NewDecoder(), b); err == nil && containsCJK(s) {
		return s
	}

	// 4. Try Shift-JIS (Japanese)
	if s, err := tryDecode(japanese.ShiftJIS.NewDecoder(), b); err == nil && containsCJK(s) {
		return s
	}

	// 5. Try Big5 (Traditional Chinese)
	if s, err := tryDecode(traditionalchinese.Big5.NewDecoder(), b); err == nil && containsCJK(s) {
		return s
	}

	// If the original was invalid UTF-8 and nothing matched CJK, try returning valid UTF-8
	if !utf8.ValidString(raw) {
		if s, err := tryDecode(simplifiedchinese.GBK.NewDecoder(), b); err == nil {
			return s
		}
		if s, err := tryDecode(japanese.ShiftJIS.NewDecoder(), b); err == nil {
			return s
		}
		if s, err := tryDecode(traditionalchinese.Big5.NewDecoder(), b); err == nil {
			return s
		}
	}

	return raw
}

func containsKana(s string) bool {
	for _, r := range s {
		if (r >= 0x3040 && r <= 0x309F) || (r >= 0x30A0 && r <= 0x30FF) {
			return true
		}
	}
	return false
}

func tryDecode(d *encoding.Decoder, b []byte) (string, error) {
	s, _, err := transform.String(d, string(b))
	if err != nil {
		return "", err
	}
	if !utf8.ValidString(s) || strings.ContainsRune(s, utf8.RuneError) {
		return "", errors.New("invalid utf8 or contains replacement char")
	}
	return s, nil
}

func containsCJK(s string) bool {
	for _, r := range s {
		if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Unified Ideographs Extension A
			(r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0x3100 && r <= 0x312F) || // Bopomofo
			(r >= 0xAC00 && r <= 0xD7AF) || // Hangul
			(r >= 0xF900 && r <= 0xFAFF) { // CJK Compatibility Ideographs
			return true
		}
	}
	return false
}

// decodeRawField checks if raw metadata is valid UTF-8. If not, it attempts
// decoding using GBK (Simplified Chinese) and Shift-JIS (Japanese).
func decodeRawField(raw string) string {
	if raw == "" {
		return ""
	}

	if !utf8.ValidString(raw) {
		// 1. Try decoding using GBK (Simplified Chinese)
		if s, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), raw); err == nil && utf8.ValidString(s) && !strings.ContainsRune(s, utf8.RuneError) {
			return decodeMetadataString(s)
		}

		// 2. Try decoding using Shift-JIS (Japanese)
		if s, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), raw); err == nil && utf8.ValidString(s) && !strings.ContainsRune(s, utf8.RuneError) {
			return decodeMetadataString(s)
		}
	}

	return decodeMetadataString(raw)
}

// Read reads metadata from the specified audio file path.
// It extracts artist, title, album, and duration (in seconds).
// If the file cannot be read, parsed, or contains no metadata, an error is returned.
func Read(filePath string) (*SongInfo, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		if errors.Is(err, tag.ErrNoTagsFound) {
			return nil, ErrNoMetadata
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	rawArtist := decodeRawField(m.Artist())
	rawTitle := decodeRawField(m.Title())
	rawAlbum := decodeRawField(m.Album())

	if DebugMode {
		printHexDebug("Artist", rawArtist)
		printHexDebug("Title", rawTitle)
		printHexDebug("Album", rawAlbum)
	}

	info := &SongInfo{
		Path:     filePath,
		Artist:   strings.TrimSpace(rawArtist),
		Title:    strings.TrimSpace(rawTitle),
		Album:    strings.TrimSpace(rawAlbum),
		Duration: extractDuration(m),
	}

	if !info.HasAnyMetadata() {
		return nil, ErrNoMetadata
	}

	return info, nil
}

// extractDuration attempts to extract track duration in seconds from the raw metadata map.
// If duration is not present or cannot be parsed, nil is returned.
func extractDuration(m tag.Metadata) *int {
	raw := m.Raw()
	if raw == nil {
		return nil
	}

	// Priority list of tag keys across ID3v2, Vorbis comment, and MP4 formats
	candidateKeys := []string{
		"TLEN", "tlen",
		"TLE", "tle",
		"length", "LENGTH", "Length",
		"duration", "DURATION", "Duration",
	}

	for _, k := range candidateKeys {
		val, exists := raw[k]
		if !exists || val == nil {
			continue
		}
		if d := parseDurationValue(val, k); d != nil {
			return d
		}
	}

	return nil
}

// parseDurationValue parses a raw metadata value into seconds.
func parseDurationValue(val interface{}, key string) *int {
	var str string
	switch v := val.(type) {
	case string:
		str = strings.TrimSpace(v)
	case fmt.Stringer:
		str = strings.TrimSpace(v.String())
	case int:
		str = strconv.Itoa(v)
	case int64:
		str = strconv.FormatInt(v, 10)
	case float64:
		str = strconv.FormatFloat(v, 'f', -1, 64)
	default:
		str = strings.TrimSpace(fmt.Sprintf("%v", v))
	}

	if str == "" {
		return nil
	}

	// Handle time strings with colons, e.g. "03:21", "00:03:21", "03:21.45"
	if strings.Contains(str, ":") {
		parts := strings.Split(str, ":")
		if len(parts) == 2 {
			mins, err1 := strconv.ParseFloat(parts[0], 64)
			secs, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil {
				total := int(math.Round(mins*60 + secs))
				if total >= 0 {
					return &total
				}
			}
		} else if len(parts) == 3 {
			hours, err1 := strconv.ParseFloat(parts[0], 64)
			mins, err2 := strconv.ParseFloat(parts[1], 64)
			secs, err3 := strconv.ParseFloat(parts[2], 64)
			if err1 == nil && err2 == nil && err3 == nil {
				total := int(math.Round(hours*3600 + mins*60 + secs))
				if total >= 0 {
					return &total
				}
			}
		}
	}

	// Handle pure numeric values
	num, err := strconv.ParseFloat(str, 64)
	if err != nil || num <= 0 {
		return nil
	}

	upperKey := strings.ToUpper(key)
	// TLEN and TLE in ID3 are standard in milliseconds
	if upperKey == "TLEN" || upperKey == "TLE" {
		if num >= 1000 {
			sec := int(math.Round(num / 1000.0))
			return &sec
		}
		sec := int(math.Round(num))
		return &sec
	}

	// For general LENGTH/DURATION tags:
	// If >= 10,000, value is almost certainly in milliseconds (10,000 ms = 10s, whereas 10,000s = 2.7h)
	if num >= 10000 {
		sec := int(math.Round(num / 1000.0))
		return &sec
	}

	sec := int(math.Round(num))
	return &sec
}
