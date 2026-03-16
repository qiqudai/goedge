//go:build !windows

package main

import "syscall"

type diskSpace struct {
	total uint64
	free  uint64
}

func getDiskSpace(path string) (diskSpace, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return diskSpace{}, err
	}
	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	free := uint64(stat.Bavail) * uint64(stat.Bsize)
	return diskSpace{total: total, free: free}, nil
}
