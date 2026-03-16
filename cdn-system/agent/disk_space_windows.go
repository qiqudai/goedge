//go:build windows

package main

import "syscall"

type diskSpace struct {
	total uint64
	free  uint64
}

func getDiskSpace(path string) (diskSpace, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return diskSpace{}, err
	}
	var freeBytes, totalBytes, _ uint64
	if err := syscall.GetDiskFreeSpaceEx(pathPtr, &freeBytes, &totalBytes, nil); err != nil {
		return diskSpace{}, err
	}
	return diskSpace{total: totalBytes, free: freeBytes}, nil
}
