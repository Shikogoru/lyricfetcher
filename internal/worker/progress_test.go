package worker

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"lyricfetcher/internal/i18n"
)

func TestProgressRenderingTTY(t *testing.T) {
	_ = i18n.Init("en")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	p := NewProgress(10, true)
	p.Update(1, 0, 0)
	p.Update(0, 1, 0)
	p.Update(0, 0, 1)
	p.Finish()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "\r") {
		t.Errorf("expected carriage return in TTY progress output, got: %q", output)
	}
	if !strings.Contains(output, "Found: 1") {
		t.Errorf("expected 'Found: 1', got: %q", output)
	}
	if !strings.Contains(output, "Not found: 1") {
		t.Errorf("expected 'Not found: 1', got: %q", output)
	}
	if !strings.Contains(output, "Errors: 1") {
		t.Errorf("expected 'Errors: 1', got: %q", output)
	}
}

func TestProgressRenderingNonTTY(t *testing.T) {
	_ = i18n.Init("en")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	p := NewProgress(10, false)
	p.Update(1, 0, 0)
	p.Finish()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if strings.Contains(output, "\r") {
		t.Errorf("expected no carriage return in non-TTY progress output, got: %q", output)
	}
}
