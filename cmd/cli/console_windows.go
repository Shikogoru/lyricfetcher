//go:build windows

package main

import "golang.org/x/sys/windows"

func init() {
	const CP_UTF8 = 65001
	_ = windows.SetConsoleOutputCP(CP_UTF8)
	_ = windows.SetConsoleCP(CP_UTF8)
}
