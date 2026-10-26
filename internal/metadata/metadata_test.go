package metadata

import (
	"errors"
	"os"
	"testing"
)

func TestParseDurationValue(t *testing.T) {
	tests := []struct {
		name     string
		val      interface{}
		key      string
		expected *int
	}{
		{
			name:     "ID3 TLEN milliseconds string",
			val:      "201000",
			key:      "TLEN",
			expected: intPtr(201),
		},
		{
			name:     "ID3 TLE milliseconds string",
			val:      "201000",
			key:      "TLE",
			expected: intPtr(201),
		},
		{
			name:     "Vorbis length in milliseconds",
			val:      "201000",
			key:      "length",
			expected: intPtr(201),
		},
		{
			name:     "Vorbis duration in seconds string",
			val:      "201",
			key:      "duration",
			expected: intPtr(201),
		},
		{
			name:     "Colon format MM:SS",
			val:      "03:21",
			key:      "duration",
			expected: intPtr(201),
		},
		{
			name:     "Colon format HH:MM:SS",
			val:      "00:03:21",
			key:      "duration",
			expected: intPtr(201),
		},
		{
			name:     "Empty value",
			val:      "",
			key:      "duration",
			expected: nil,
		},
		{
			name:     "Invalid string",
			val:      "not_a_number",
			key:      "duration",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDurationValue(tc.val, tc.key)
			if tc.expected == nil {
				if got != nil {
					t.Fatalf("expected nil, got %d", *got)
				}
			} else {
				if got == nil {
					t.Fatalf("expected %d, got nil", *tc.expected)
				}
				if *got != *tc.expected {
					t.Fatalf("expected %d, got %d", *tc.expected, *got)
				}
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	t.Run("Split empty artist with hyphen", func(t *testing.T) {
		orig := &SongInfo{
			Path:   "test.mp3",
			Title:  "夜宴风波-泠鸢yousa",
			Artist: "",
		}
		norm := Normalize(orig)
		if norm.Title != "夜宴风波" {
			t.Errorf("expected title '夜宴风波', got '%s'", norm.Title)
		}
		if norm.Artist != "泠鸢yousa" {
			t.Errorf("expected artist '泠鸢yousa', got '%s'", norm.Artist)
		}
		// Check that input is not modified
		if orig.Artist != "" || orig.Title != "夜宴风波-泠鸢yousa" {
			t.Errorf("original input was modified: %+v", orig)
		}
	})

	t.Run("Split on LAST dash", func(t *testing.T) {
		orig := &SongInfo{
			Path:   "test.mp3",
			Title:  "Part 1 - Part 2 - Band Name",
			Artist: "",
		}
		norm := Normalize(orig)
		if norm.Title != "Part 1 - Part 2" {
			t.Errorf("expected title 'Part 1 - Part 2', got '%s'", norm.Title)
		}
		if norm.Artist != "Band Name" {
			t.Errorf("expected artist 'Band Name', got '%s'", norm.Artist)
		}
	})

	t.Run("Split with en-dash and em-dash", func(t *testing.T) {
		origEn := &SongInfo{Title: "Title–Artist", Artist: ""}
		normEn := Normalize(origEn)
		if normEn.Title != "Title" || normEn.Artist != "Artist" {
			t.Errorf("en-dash split failed: title=%s artist=%s", normEn.Title, normEn.Artist)
		}

		origEm := &SongInfo{Title: "Title—Artist", Artist: ""}
		normEm := Normalize(origEm)
		if normEm.Title != "Title" || normEm.Artist != "Artist" {
			t.Errorf("em-dash split failed: title=%s artist=%s", normEm.Title, normEm.Artist)
		}
	})

	t.Run("Full metadata unchanged", func(t *testing.T) {
		dur := 201
		orig := &SongInfo{
			Path:     "01. Drive By.mp3",
			Title:    "Drive By",
			Artist:   "Train",
			Album:    "California 37",
			Duration: &dur,
		}
		norm := Normalize(orig)
		if norm.Title != "Drive By" || norm.Artist != "Train" || norm.Album != "California 37" {
			t.Errorf("unexpected change: %+v", norm)
		}
		if !orig.Equal(norm) {
			t.Errorf("expected Equal to be true")
		}
	})

	t.Run("Remove trailing artist suffix from title", func(t *testing.T) {
		orig := &SongInfo{
			Title:  "Song Name - Train",
			Artist: "Train",
			Album:  "Some Album",
		}
		norm := Normalize(orig)
		if norm.Title != "Song Name" {
			t.Errorf("expected title 'Song Name', got '%s'", norm.Title)
		}
		if norm.Artist != "Train" {
			t.Errorf("expected artist 'Train', got '%s'", norm.Artist)
		}
	})

	t.Run("Remove leading artist prefix from title", func(t *testing.T) {
		orig := &SongInfo{
			Title:  "Train - Song Name",
			Artist: "Train",
			Album:  "Some Album",
		}
		norm := Normalize(orig)
		if norm.Title != "Song Name" {
			t.Errorf("expected title 'Song Name', got '%s'", norm.Title)
		}
		if norm.Artist != "Train" {
			t.Errorf("expected artist 'Train', got '%s'", norm.Artist)
		}
	})

	t.Run("Trim whitespace and Unicode NFC normalization", func(t *testing.T) {
		// Decomposed 'e' + combining acute accent U+0301 (NFD)
		decomposed := "Cafe\u0301"
		orig := &SongInfo{
			Title:  "  " + decomposed + "  ",
			Artist: "  Artist  ",
			Album:  "  Album  ",
		}
		norm := Normalize(orig)
		expectedTitle := "Café" // NFC
		if norm.Title != expectedTitle {
			t.Errorf("expected NFC title '%s', got '%s'", expectedTitle, norm.Title)
		}
		if norm.Artist != "Artist" {
			t.Errorf("expected trimmed artist 'Artist', got '%s'", norm.Artist)
		}
		if norm.Album != "Album" {
			t.Errorf("expected trimmed album 'Album', got '%s'", norm.Album)
		}
	})
}

func TestReadAudioFiles(t *testing.T) {
	if _, err := os.Stat("../../test_audio/01. Drive By.flac"); os.IsNotExist(err) {
		t.Skip("skipping file-based audio test: test_audio directory not present")
	}

	// 1. MP3 with full metadata
	infoMP3, err := Read("../../test_audio/California 37/01. Drive By.mp3")
	if err != nil {
		t.Fatalf("failed to read MP3: %v", err)
	}
	if infoMP3.Artist != "Train" {
		t.Errorf("expected artist 'Train', got '%s'", infoMP3.Artist)
	}
	if infoMP3.Title != "Drive By" {
		t.Errorf("expected title 'Drive By', got '%s'", infoMP3.Title)
	}
	if infoMP3.Album != "California 37" {
		t.Errorf("expected album 'California 37', got '%s'", infoMP3.Album)
	}
	if infoMP3.Duration == nil || *infoMP3.Duration != 196 {
		t.Errorf("expected duration 196, got %v", infoMP3.Duration)
	}

	// 2. FLAC with full metadata
	infoFLAC, err := Read("../../test_audio/01. Drive By.flac")
	if err != nil {
		t.Fatalf("failed to read FLAC: %v", err)
	}
	if infoFLAC.Artist != "Train" {
		t.Errorf("expected artist 'Train', got '%s'", infoFLAC.Artist)
	}
	if infoFLAC.Title != "Drive By" {
		t.Errorf("expected title 'Drive By', got '%s'", infoFLAC.Title)
	}
	if infoFLAC.Album != "California 37" {
		t.Errorf("expected album 'California 37', got '%s'", infoFLAC.Album)
	}
	if infoFLAC.Duration == nil || *infoFLAC.Duration != 196 {
		t.Errorf("expected duration 196, got %v", infoFLAC.Duration)
	}

	// 3. Partial metadata MP3
	infoPartial, err := Read("../../test_audio/partial.mp3")
	if err != nil {
		t.Fatalf("failed to read partial MP3: %v", err)
	}
	if infoPartial.Artist != "Train" {
		t.Errorf("expected artist 'Train', got '%s'", infoPartial.Artist)
	}
	if infoPartial.Title != "Drive By" {
		t.Errorf("expected title 'Drive By', got '%s'", infoPartial.Title)
	}
	if infoPartial.Album != "" {
		t.Errorf("expected empty album, got '%s'", infoPartial.Album)
	}
	if infoPartial.Duration != nil {
		t.Errorf("expected nil duration, got %v", infoPartial.Duration)
	}

	// 4. No tags MP3
	_, errNoTags := Read("../../test_audio/no_tags.mp3")
	if errNoTags == nil {
		t.Fatalf("expected error for file without tags, got nil")
	}
	if !errors.Is(errNoTags, ErrNoMetadata) {
		t.Logf("no tags returned error: %v", errNoTags)
	}

	// 5. Corrupt file
	_, errCorrupt := Read("../../test_audio/corrupt.mp3")
	if errCorrupt == nil {
		t.Fatalf("expected error for corrupt file, got nil")
	}
}

func intPtr(i int) *int {
	return &i
}
