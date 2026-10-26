package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lyricfetcher/internal/metadata"
)

const (
	lrclibGetURL    = "https://lrclib.net/api/get"
	lrclibSearchURL = "https://lrclib.net/api/search"
	lrclibUserAgent = "LyricFetcher/0.3 (+https://github.com/lyricfetcher/lyricfetcher)"
)

type lrclibResponse struct {
	ID           int     `json:"id"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  *string `json:"plainLyrics"`
	SyncedLyrics *string `json:"syncedLyrics"`
}

type LRCLIBProvider struct {
	client *http.Client
}

func NewLRCLIBProvider() *LRCLIBProvider {
	return &LRCLIBProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *LRCLIBProvider) Name() string {
	return "lrclib"
}

func (p *LRCLIBProvider) Fetch(info *metadata.SongInfo, strict bool) (*Lyric, error) {
	if info == nil {
		return nil, nil
	}

	title := strings.TrimSpace(info.Title)
	artist := strings.TrimSpace(info.Artist)
	album := strings.TrimSpace(info.Album)

	// 4. Nếu track_name rỗng, thử lấy tên file (bỏ đuôi .flac/.mp3) làm track_name
	if title == "" && info.Path != "" {
		title = strings.TrimSuffix(filepath.Base(info.Path), filepath.Ext(info.Path))
	}

	if title == "" {
		return nil, nil
	}

	// 2. Đảm bảo track_name KHÔNG chứa artist. Nếu Title có dạng "Tên bài - Ca sĩ", phải tách ra trước khi gửi
	for _, dash := range []string{" - ", " – ", " — ", "-", "–", "—"} {
		if idx := strings.LastIndex(title, dash); idx > 0 {
			before := strings.TrimSpace(title[:idx])
			after := strings.TrimSpace(title[idx+len(dash):])
			if before != "" && after != "" {
				if artist == "" {
					title = before
					artist = after
				} else if strings.EqualFold(after, artist) {
					title = before
				}
				break
			}
		}
	}
	if artist != "" {
		for _, dash := range []string{" - ", " – ", " — ", "-", "–", "—"} {
			prefix := artist + dash
			if strings.HasPrefix(title, prefix) {
				title = strings.TrimSpace(strings.TrimPrefix(title, prefix))
				break
			}
		}
	}

	hasArtist := artist != ""
	hasAlbum := album != ""
	hasDuration := info.Duration != nil

	type tierQuery struct {
		isSearch bool
		params   url.Values
	}

	var tiers []tierQuery

	// 3. Nếu artist_name rỗng, chuyển sang gọi /api/search?q=<title> thay vì /api/get
	if hasArtist {
		// Tier 1: track_name + artist_name + album + duration (nếu có)
		if hasAlbum && hasDuration {
			v1 := url.Values{}
			v1.Set("track_name", title)
			v1.Set("artist_name", artist)
			v1.Set("album_name", album)
			v1.Set("duration", strconv.Itoa(*info.Duration))
			tiers = append(tiers, tierQuery{isSearch: false, params: v1})
		}

		// Tier 2: track_name + artist_name (hai tham số riêng biệt)
		v2 := url.Values{}
		v2.Set("track_name", title)
		v2.Set("artist_name", artist)
		tiers = append(tiers, tierQuery{isSearch: false, params: v2})

		// Tier 3 (Fallback nếu /api/get không ra): /api/search?q=<title>
		v3 := url.Values{}
		v3.Set("q", title)
		tiers = append(tiers, tierQuery{isSearch: true, params: v3})
	} else {
		// artist_name rỗng -> chuyển sang gọi /api/search?q=<title>
		v3 := url.Values{}
		v3.Set("q", title)
		tiers = append(tiers, tierQuery{isSearch: true, params: v3})
	}

	for i, tier := range tiers {
		if i > 0 {
			time.Sleep(300 * time.Millisecond)
		}

		var targetURL string
		if tier.isSearch {
			targetURL = lrclibSearchURL + "?" + tier.params.Encode()
		} else {
			targetURL = lrclibGetURL + "?" + tier.params.Encode()
		}

		// 1. Log ra URL request thực tế (dùng log.Printf("[DEBUG] LRCLIB URL: %s", url))
		log.Printf("[DEBUG] LRCLIB URL: %s", targetURL)
		debugLog("[DEBUG] LRCLIB URL: %s", targetURL)

		respData, statusCode, err := p.executeRequestWithRetry(targetURL)
		if err != nil || statusCode >= 500 {
			// If /api/get encounters 5xx (e.g. 503 Server Overloaded), fallback to /api/search with same params
			if !tier.isSearch {
				searchURL := lrclibSearchURL + "?" + tier.params.Encode()
				log.Printf("[DEBUG] LRCLIB URL: %s (5xx fallback)", searchURL)
				debugLog("[DEBUG] LRCLIB URL: %s (5xx fallback)", searchURL)
				sData, sCode, sErr := p.executeRequestWithRetry(searchURL)
				if sErr == nil && sCode == http.StatusOK {
					respData = sData
					statusCode = sCode
					tier.isSearch = true
					err = nil
				}
			}
		}

		if err != nil {
			return nil, err
		}

		if statusCode == http.StatusNotFound {
			debugLog("[DEBUG] LRCLIB response: %d, results: 0", statusCode)
			continue
		}

		if statusCode >= 500 {
			debugLog("[DEBUG] LRCLIB response: %d", statusCode)
			return nil, fmt.Errorf("lrclib server error: status %d", statusCode)
		}

		if statusCode != http.StatusOK {
			debugLog("[DEBUG] LRCLIB response: %d", statusCode)
			continue
		}

		var candidates []lrclibResponse
		if tier.isSearch {
			if err := json.Unmarshal(respData, &candidates); err != nil {
				continue
			}
		} else {
			var single lrclibResponse
			if err := json.Unmarshal(respData, &single); err != nil {
				continue
			}
			candidates = append(candidates, single)
		}

		debugLog("[DEBUG] LRCLIB response: %d, results: %d", statusCode, len(candidates))

		if tier.isSearch {
			// Prioritize high-confidence match (e.g. duration matches within 2s)
			var firstValidLyric *Lyric
			for _, item := range candidates {
				lyric := p.parseCandidate(info, &item)
				if lyric == nil {
					continue
				}
				if lyric.Confidence == "high" {
					return lyric, nil
				}
				if firstValidLyric == nil && !strict {
					firstValidLyric = lyric
				}
			}
			if firstValidLyric != nil {
				return firstValidLyric, nil
			}
		} else {
			for _, item := range candidates {
				lyric := p.parseCandidate(info, &item)
				if lyric == nil {
					continue
				}
				if strict && lyric.Confidence != "high" {
					continue
				}
				return lyric, nil
			}
		}
	}

	return nil, nil
}

func (p *LRCLIBProvider) executeRequestWithRetry(targetURL string) ([]byte, int, error) {
	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", lrclibUserAgent)

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, 0, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			retryAfterSec := 1
			if h := resp.Header.Get("Retry-After"); h != "" {
				if s, err := strconv.Atoi(h); err == nil && s > 0 {
					retryAfterSec = s
				}
			}
			if attempt < maxRetries {
				time.Sleep(time.Duration(retryAfterSec) * time.Second)
				continue
			}
			return nil, http.StatusTooManyRequests, fmt.Errorf("rate limited by lrclib (429)")
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, resp.StatusCode, err
		}

		return body, resp.StatusCode, nil
	}

	return nil, 0, fmt.Errorf("exceeded max retries")
}

func (p *LRCLIBProvider) parseCandidate(info *metadata.SongInfo, resp *lrclibResponse) *Lyric {
	if resp.Instrumental {
		return &Lyric{
			Content:      "",
			Synced:       false,
			Source:       "lrclib",
			Confidence:   "high",
			Instrumental: true,
		}
	}

	var content string
	var synced bool

	if resp.SyncedLyrics != nil && strings.TrimSpace(*resp.SyncedLyrics) != "" {
		content = *resp.SyncedLyrics
		synced = true
	} else if resp.PlainLyrics != nil && strings.TrimSpace(*resp.PlainLyrics) != "" {
		content = *resp.PlainLyrics
		synced = false
	} else {
		return nil
	}

	confidence := "low"
	if info.Duration != nil && resp.Duration > 0 && math.Abs(float64(*info.Duration)-resp.Duration) <= 2 {
		confidence = "high"
	}

	return &Lyric{
		Content:      content,
		Synced:       synced,
		Source:       "lrclib",
		Confidence:   confidence,
		Instrumental: false,
	}
}
