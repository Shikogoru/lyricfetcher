//go:build windows

package console

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleCP    = kernel32.NewProc("SetConsoleCP")
	procSetConsoleOutCP = kernel32.NewProc("SetConsoleOutputCP")
	procWriteConsoleW   = kernel32.NewProc("WriteConsoleW")

	mu sync.Mutex
)

func init() {
	FixEncoding()
}

// FixEncoding ensures that the Windows console code page is UTF-8 (65001).
func FixEncoding() {
	procSetConsoleCP.Call(65001)
	procSetConsoleOutCP.Call(65001)
}

func isConsole(f *os.File) bool {
	if f == nil {
		return false
	}
	var mode uint32
	err := syscall.GetConsoleMode(syscall.Handle(f.Fd()), &mode)
	return err == nil
}

func writeString(f *os.File, s string) (int, error) {
	if s == "" {
		return 0, nil
	}

	mu.Lock()
	defer mu.Unlock()

	if !isConsole(f) {
		return f.WriteString(s)
	}

	wchars := utf16.Encode([]rune(s))
	if len(wchars) == 0 {
		return 0, nil
	}

	var written uint32
	r1, _, _ := procWriteConsoleW.Call(
		f.Fd(),
		uintptr(unsafe.Pointer(&wchars[0])),
		uintptr(len(wchars)),
		uintptr(unsafe.Pointer(&written)),
		0,
	)
	if r1 == 0 {
		return f.WriteString(s)
	}
	return len(s), nil
}

// Print outputs to standard output using Unicode-safe console writes.
func Print(a ...interface{}) (int, error) {
	return writeString(os.Stdout, fmt.Sprint(a...))
}

// Println outputs to standard output followed by a newline.
func Println(a ...interface{}) (int, error) {
	return writeString(os.Stdout, fmt.Sprintln(a...))
}

// Printf formats and outputs to standard output.
func Printf(format string, a ...interface{}) (int, error) {
	return writeString(os.Stdout, fmt.Sprintf(format, a...))
}

// WriteString writes a raw string to standard output.
func WriteString(s string) (int, error) {
	return writeString(os.Stdout, s)
}

// Eprintf formats and outputs to standard error.
func Eprintf(format string, a ...interface{}) (int, error) {
	return writeString(os.Stderr, fmt.Sprintf(format, a...))
}

// Eprintln outputs to standard error followed by a newline.
func Eprintln(a ...interface{}) (int, error) {
	return writeString(os.Stderr, fmt.Sprintln(a...))
}
