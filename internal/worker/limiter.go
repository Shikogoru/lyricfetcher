package worker

import (
	"context"
	"sync"
	"time"
)

// LRCLIBLimiter enforces a maximum rate of reqPerSec requests per second.
type LRCLIBLimiter struct {
	ticker *time.Ticker
	tokens chan struct{}
}

func NewLRCLIBLimiter(ratePerSec int) *LRCLIBLimiter {
	if ratePerSec <= 0 {
		ratePerSec = 4
	}
	interval := time.Second / time.Duration(ratePerSec)
	l := &LRCLIBLimiter{
		ticker: time.NewTicker(interval),
		tokens: make(chan struct{}, ratePerSec),
	}
	for i := 0; i < ratePerSec; i++ {
		l.tokens <- struct{}{}
	}
	go func() {
		for range l.ticker.C {
			select {
			case l.tokens <- struct{}{}:
			default:
			}
		}
	}()
	return l
}

func (l *LRCLIBLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-l.tokens:
		return nil
	}
}

func (l *LRCLIBLimiter) Stop() {
	l.ticker.Stop()
}

// NetEaseLimiter enforces concurrency = 1 and max 1 req/sec.
type NetEaseLimiter struct {
	sem         chan struct{}
	mu          sync.Mutex
	lastRequest time.Time
	minInterval time.Duration
}

func NewNetEaseLimiter() *NetEaseLimiter {
	sem := make(chan struct{}, 1)
	sem <- struct{}{}
	return &NetEaseLimiter{
		sem:         sem,
		minInterval: 1 * time.Second,
	}
}

func (l *NetEaseLimiter) Acquire(ctx context.Context) (func(), error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.sem:
	}

	l.mu.Lock()
	elapsed := time.Since(l.lastRequest)
	if elapsed < l.minInterval {
		wait := l.minInterval - elapsed
		l.mu.Unlock()
		select {
		case <-ctx.Done():
			l.sem <- struct{}{}
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		l.mu.Lock()
	}
	l.lastRequest = time.Now()
	l.mu.Unlock()

	return func() {
		l.sem <- struct{}{}
	}, nil
}
