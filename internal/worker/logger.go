package worker

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileLogger struct {
	mu   sync.Mutex
	file *os.File
	path string
}

func NewFileLogger(path string) (*FileLogger, error) {
	if path == "" {
		path = "lyricfetcher.log"
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &FileLogger{
		file: f,
		path: path,
	}, nil
}

func (l *FileLogger) Log(level, workerTag, format string, args ...any) {
	if l == nil || l.file == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("%s [%s] [%s] %s\n", timestamp, level, workerTag, msg)
	_, _ = l.file.WriteString(line)
}

func (l *FileLogger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *FileLogger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}
