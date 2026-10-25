//go:build windows

package main

import "golang.org/x/sys/windows"

func diskFree(p string) uint64 {
	if p == "" {
		return 0
	}
	ptr, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return 0
	}
	var free, total, totalFree uint64
	if windows.GetDiskFreeSpaceEx(ptr, &free, &total, &totalFree) != nil {
		return 0
	}
	return free
}
