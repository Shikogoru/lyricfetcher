//go:build !windows

package console

import (
	"fmt"
	"os"
)

// FixEncoding is a no-op on non-Windows platforms.
func FixEncoding() {}

// Print outputs to standard output.
func Print(a ...interface{}) (int, error) {
	return fmt.Print(a...)
}

// Println outputs to standard output followed by a newline.
func Println(a ...interface{}) (int, error) {
	return fmt.Println(a...)
}

// Printf formats and outputs to standard output.
func Printf(format string, a ...interface{}) (int, error) {
	return fmt.Printf(format, a...)
}

// WriteString writes a raw string to standard output.
func WriteString(s string) (int, error) {
	return os.Stdout.WriteString(s)
}

// Eprintf formats and outputs to standard error.
func Eprintf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(os.Stderr, format, a...)
}

// Eprintln outputs to standard error followed by a newline.
func Eprintln(a ...interface{}) (int, error) {
	return fmt.Fprintln(os.Stderr, a...)
}
