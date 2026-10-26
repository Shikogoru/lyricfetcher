package worker

import (
	"os"
	"strings"
	"sync"

	"lyricfetcher/internal/console"
	"lyricfetcher/internal/i18n"
)

type Progress struct {
	mu        sync.Mutex
	total     int
	processed int
	found     int
	notFound  int
	errors    int
	isTTY     bool
}

func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func NewProgress(total int, isTTY bool) *Progress {
	p := &Progress{
		total: total,
		isTTY: isTTY,
	}
	if isTTY && total > 0 {
		p.render()
	}
	return p
}

func (p *Progress) SetTTY(isTTY bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isTTY = isTTY
}

func (p *Progress) Update(foundDelta, notFoundDelta, errorsDelta int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.processed++
	p.found += foundDelta
	p.notFound += notFoundDelta
	p.errors += errorsDelta

	if p.isTTY {
		p.render()
	}
}

// PrintFileBlock prints a completed file output block in non-TTY mode atomically.
// In TTY mode, individual file blocks are suppressed so the progress bar remains clean.
func (p *Progress) PrintFileBlock(block string) {
	if block == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isTTY {
		console.WriteString(block)
	}
}

func (p *Progress) render() {
	barWidth := 20
	percent := 0
	filled := 0
	if p.total > 0 {
		percent = int(float64(p.processed) / float64(p.total) * 100)
		filled = int(float64(p.processed) / float64(p.total) * float64(barWidth))
	}
	if filled > barWidth {
		filled = barWidth
	}

	var bar string
	if filled == barWidth {
		bar = strings.Repeat("=", barWidth)
	} else if filled > 0 {
		bar = strings.Repeat("=", filled-1) + ">" + strings.Repeat(" ", barWidth-filled)
	} else {
		bar = strings.Repeat(" ", barWidth)
	}

	line := i18n.Tf("progress_line", bar, p.processed, p.total, percent, p.found, p.notFound, p.errors)
	console.Printf("\r%-80s", line)
}

func (p *Progress) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isTTY && p.total > 0 {
		console.Println()
	}
}
