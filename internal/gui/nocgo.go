//go:build !cgo

package gui

import (
	"fmt"
	"os"
)

// HasGUI returns false when built without CGO.
func HasGUI() bool {
	return false
}

// Run outputs an error and exits when built without CGO.
func Run() {
	fmt.Fprintln(os.Stderr, "GUI is not supported in this build (compiled without CGO). Please use the Windows amd64 build.")
	os.Exit(1)
}
