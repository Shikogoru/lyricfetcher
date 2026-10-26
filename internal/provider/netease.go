package provider

// NetEase API is unofficial. Use as fallback only.
// Respect rate limits and terms of service.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"lyricfetcher/internal/metadata"
)

const (
	neteaseCloudSearchURL = "https://music.163.com/api/cloudsearch/pc"
	neteaseLyricURL       = "https://music.163.com/api/song/lyric"
	neteaseUserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	neteaseReferer        = "https://music.163.com/"
	neteaseOrigin         = "https://music.163.com"
	neteaseCookie         = "os=pc; appver=2.9.7"
	defaultSpoofIP        = "118.89.204.198"
)

type NetEaseOptions struct {
	SpoofIP      bool
	SpoofIPValue string
}

type neteaseSongItem struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Duration int64  `json:"duration"` // ms
}

type neteaseSearchResponse struct {
	Result struct {
		Songs     []neteaseSongItem `json:"songs"`
		SongCount int               `json:"songCount"`
	} `json:"result"`
	Code int `json:"code"`
}

type neteaseLyricResponse struct {
	Lrc struct {
		Lyric string `json:"lyric"`
	} `json:"lrc"`
	Tlyric struct {
		Lyric string `json:"lyric"`
	} `json:"tlyric"`
	Code int `json:"code"`
}

type NetEaseProvider struct {
	client *http.Client
	opts   NetEaseOptions
}

func NewNetEaseProvider(opts NetEaseOptions) *NetEaseProvider {
	if opts.SpoofIPValue == "" {
		opts.SpoofIPValue = defaultSpoofIP
	}
	return &NetEaseProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		opts: opts,
	}
}

func (p *NetEaseProvider) Name() string {
	return "netease"
}

func (p *NetEaseProvider) search(query string) (*neteaseSearchResponse, error) {
	v := url.Values{}
	v.Set("s", query)
	v.Set("type", "1")
	v.Set("limit", "5")
	v.Set("offset", "0")

	debugLog("[DEBUG] NetEase POST: %s (query=%s)", neteaseCloudSearchURL, query)

	req, err := http.NewRequest(http.MethodPost, neteaseCloudSearchURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", neteaseUserAgent)
	req.Header.Set("Referer", neteaseReferer)
	req.Header.Set("Origin", neteaseOrigin)
	req.Header.Set("Cookie", neteaseCookie)
	if p.opts.SpoofIP {
		req.Header.Set("X-Real-IP", p.opts.SpoofIPValue)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		debugLog("[DEBUG] NetEase response: %d", resp.StatusCode)
		return nil, fmt.Errorf("netease server error: status %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		debugLog("[DEBUG] NetEase response: %d", resp.StatusCode)
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchResult neteaseSearchResponse
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, nil
	}

	debugLog("[DEBUG] NetEase response: %d, songs found: %d", searchResult.Code, len(searchResult.Result.Songs))

	if searchResult.Code == 200 && len(searchResult.Result.Songs) == 0 {
		log.Printf("[DEBUG] NetEase returned empty (possible bot detection)")
		debugLog("[DEBUG] NetEase returned empty (possible bot detection)")
	}

	return &searchResult, nil
}

func (p *NetEaseProvider) Fetch(info *metadata.SongInfo, strict bool) (*Lyric, error) {
	if strings.TrimSpace(info.Title) == "" {
		return nil, nil
	}

	title := strings.TrimSpace(info.Title)
	artist := strings.TrimSpace(info.Artist)

	query := title
	if artist != "" {
		query = title + " " + artist
	}

	// Step 1: Search song ID
	searchResult, err := p.search(query)
	if err != nil {
		return nil, err
	}

	// Fallback: If searching "Title Artist" returned 0 songs, try searching Title alone
	if (searchResult == nil || len(searchResult.Result.Songs) == 0) && artist != "" {
		fallbackResult, fbErr := p.search(title)
		if fbErr == nil && fallbackResult != nil && len(fallbackResult.Result.Songs) > 0 {
			searchResult = fallbackResult
		}
	}

	if searchResult == nil || len(searchResult.Result.Songs) == 0 {
		return nil, nil
	}

	var bestSong *neteaseSongItem
	confidence := "low"

	// Match duration within 2s if known
	if info.Duration != nil {
		for i := range searchResult.Result.Songs {
			s := &searchResult.Result.Songs[i]
			if s.Duration > 0 {
				durSec := float64(s.Duration) / 1000.0
				if math.Abs(float64(*info.Duration)-durSec) <= 2 {
					bestSong = s
					confidence = "high"
					break
				}
			}
		}
	}

	if bestSong == nil {
		bestSong = &searchResult.Result.Songs[0]
		if info.Duration == nil {
			// When duration is not known in metadata, matching by search query gives high confidence
			confidence = "high"
		} else {
			confidence = "low"
		}
	}

	if strict && confidence != "high" {
		return nil, nil
	}

	// Step 2 & 3: Fetch lyric by song ID
	lv := url.Values{}
	lv.Set("id", strconv.FormatInt(bestSong.ID, 10))
	lv.Set("lv", "-1")
	lv.Set("kv", "-1")
	lv.Set("tv", "-1")

	lyricReqURL := neteaseLyricURL + "?" + lv.Encode()
	debugLog("[DEBUG] NetEase GET: %s", lyricReqURL)

	lReq, err := http.NewRequest(http.MethodGet, lyricReqURL, nil)
	if err != nil {
		return nil, err
	}
	lReq.Header.Set("User-Agent", neteaseUserAgent)
	lReq.Header.Set("Referer", neteaseReferer)
	lReq.Header.Set("Origin", neteaseOrigin)
	lReq.Header.Set("Cookie", neteaseCookie)
	if p.opts.SpoofIP {
		lReq.Header.Set("X-Real-IP", p.opts.SpoofIPValue)
	}

	lResp, err := p.client.Do(lReq)
	if err != nil {
		return nil, err
	}
	defer lResp.Body.Close()

	if lResp.StatusCode >= 500 {
		debugLog("[DEBUG] NetEase lyric response: %d", lResp.StatusCode)
		return nil, fmt.Errorf("netease server error: status %d", lResp.StatusCode)
	}
	if lResp.StatusCode != http.StatusOK {
		debugLog("[DEBUG] NetEase lyric response: %d", lResp.StatusCode)
		return nil, nil
	}

	lBody, err := io.ReadAll(lResp.Body)
	if err != nil {
		return nil, err
	}

	var lyricData neteaseLyricResponse
	if err := json.Unmarshal(lBody, &lyricData); err != nil {
		return nil, nil
	}

	debugLog("[DEBUG] NetEase lyric response: %d", lyricData.Code)

	content := strings.TrimSpace(lyricData.Lrc.Lyric)
	if content == "" {
		content = strings.TrimSpace(lyricData.Tlyric.Lyric)
	}

	if content == "" {
		return nil, nil
	}

	synced := strings.Contains(content, "[") && strings.Contains(content, ":")

	return &Lyric{
		Content:      content,
		Synced:       synced,
		Source:       "netease",
		Confidence:   confidence,
		Instrumental: false,
	}, nil
}
