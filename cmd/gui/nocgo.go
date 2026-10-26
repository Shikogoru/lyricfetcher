//go:build !cgo

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Error: LyricFetcher GUI requires CGO to be enabled. Please build with CGO_ENABLED=1 and a C compiler (GCC/Clang).")
	os.Exit(1)
}
