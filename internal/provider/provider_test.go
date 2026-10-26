package provider

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"lyricfetcher/internal/metadata"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestParseCandidate(t *testing.T) {
	p := NewLRCLIBProvider()

	dur := 200
	info := &metadata.SongInfo{
		Title:    "Test Song",
		Artist:   "Test Artist",
		Duration: &dur,
	}

	// 1. Instrumental
	instResp := &lrclibResponse{
		Instrumental: true,
	}
	lyricInst := p.parseCandidate(info, instResp)
	if !lyricInst.Instrumental || lyricInst.Confidence != "high" {
		t.Errorf("expected instrumental high confidence, got %+v", lyricInst)
	}

	// 2. Synced lyrics with matching duration
	synced := "[00:01.00] Hello"
	matchResp := &lrclibResponse{
		Duration:     201.0,
		SyncedLyrics: &synced,
	}
	lyricMatch := p.parseCandidate(info, matchResp)
	if lyricMatch.Confidence != "high" || !lyricMatch.Synced || lyricMatch.Content != synced {
		t.Errorf("expected synced high confidence, got %+v", lyricMatch)
	}

	// 3. Plain lyrics with mismatched duration (> 2s difference)
	plain := "Hello world"
	mismatchResp := &lrclibResponse{
		Duration:    190.0,
		PlainLyrics: &plain,
	}
	lyricMismatch := p.parseCandidate(info, mismatchResp)
	if lyricMismatch.Confidence != "low" || lyricMismatch.Synced || lyricMismatch.Content != plain {
		t.Errorf("expected plain low confidence, got %+v", lyricMismatch)
	}
}

func TestLRCLIBFallbackToSearch(t *testing.T) {
	p := NewLRCLIBProvider()
	dur := 180
	info := &metadata.SongInfo{
		Title:    "Rare Song",
		Artist:   "Collab Artist 1 & 2",
		Duration: &dur,
	}

	var getCalled, searchCalled bool

	p.client.Transport = roundTripFunc(func(req *http.Request) *http.Response {
		if strings.Contains(req.URL.Path, "/api/get") {
			getCalled = true
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": "Not Found"}`)),
				Header:     make(http.Header),
			}
		}
		if strings.Contains(req.URL.Path, "/api/search") {
			searchCalled = true
			jsonResp := `[
				{
					"id": 999,
					"trackName": "Rare Song",
					"artistName": "Collab Artist 1",
					"duration": 181.0,
					"syncedLyrics": "[00:01.00] Fallback line"
				}
			]`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}
		}
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString("")),
			Header:     make(http.Header),
		}
	})

	lyric, err := p.Fetch(info, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !getCalled {
		t.Errorf("expected /api/get to be called first")
	}
	if !searchCalled {
		t.Errorf("expected fallback /api/search to be called")
	}
	if lyric == nil || lyric.Content != "[00:01.00] Fallback line" {
		t.Fatalf("expected fallback lyric, got: %+v", lyric)
	}
	if lyric.Confidence != "high" {
		t.Errorf("expected high confidence for duration match, got: %s", lyric.Confidence)
	}
}

func TestNetEaseCloudSearchAndHeaders(t *testing.T) {
	opts := NetEaseOptions{
		SpoofIP: false,
	}
	p := NewNetEaseProvider(opts)

	dur := 210
	info := &metadata.SongInfo{
		Title:    "Cloud Song",
		Artist:   "NetEase Artist",
		Duration: &dur,
	}

	var postSearchCalled, getLyricCalled bool

	p.client.Transport = roundTripFunc(func(req *http.Request) *http.Response {
		if strings.Contains(req.URL.Path, "/api/cloudsearch/pc") {
			postSearchCalled = true
			if req.Method != http.MethodPost {
				t.Errorf("expected POST method, got: %s", req.Method)
			}
			if req.Header.Get("Referer") != neteaseReferer {
				t.Errorf("expected Referer header '%s', got '%s'", neteaseReferer, req.Header.Get("Referer"))
			}
			if req.Header.Get("Origin") != neteaseOrigin {
				t.Errorf("expected Origin header '%s', got '%s'", neteaseOrigin, req.Header.Get("Origin"))
			}
			if req.Header.Get("Cookie") != neteaseCookie {
				t.Errorf("expected Cookie header '%s', got '%s'", neteaseCookie, req.Header.Get("Cookie"))
			}
			if req.Header.Get("X-Real-IP") != "" {
				t.Errorf("expected no X-Real-IP when SpoofIP is false, got '%s'", req.Header.Get("X-Real-IP"))
			}

			jsonResp := `{
				"code": 200,
				"result": {
					"songs": [
						{"id": 88888, "name": "Cloud Song", "duration": 210000}
					],
					"songCount": 1
				}
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}
		}

		if strings.Contains(req.URL.Path, "/api/song/lyric") {
			getLyricCalled = true
			if req.URL.Query().Get("id") != "88888" {
				t.Errorf("expected lyric song id 88888, got: %s", req.URL.Query().Get("id"))
			}
			jsonResp := `{
				"code": 200,
				"lrc": {
					"lyric": "[00:02.00] NetEase line"
				}
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
				Header:     make(http.Header),
			}
		}

		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString("")),
			Header:     make(http.Header),
		}
	})

	lyric, err := p.Fetch(info, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !postSearchCalled {
		t.Errorf("expected POST cloudsearch to be called")
	}
	if !getLyricCalled {
		t.Errorf("expected GET lyric to be called")
	}
	if lyric == nil || lyric.Content != "[00:02.00] NetEase line" {
		t.Fatalf("expected netease lyric, got: %+v", lyric)
	}
	if lyric.Source != "netease" {
		t.Errorf("expected Source=netease, got: %s", lyric.Source)
	}
	if !lyric.Synced {
		t.Errorf("expected Synced=true")
	}
}
