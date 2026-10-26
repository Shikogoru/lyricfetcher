//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

func isStartedByExplorer() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleProcessList := kernel32.NewProc("GetConsoleProcessList")
	var pids [2]uint32
	r, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), 2)
	// If only 1 process is attached, Windows spawned a temporary console exclusively for us (Explorer double-click)
	return r == 1
}

func hideConsoleIfExplorer() {
	if isStartedByExplorer() {
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		user32 := syscall.NewLazyDLL("user32.dll")
		hwnd, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
		if hwnd != 0 {
			user32.NewProc("ShowWindow").Call(hwnd, 0) // SW_HIDE
		}
	}
}
