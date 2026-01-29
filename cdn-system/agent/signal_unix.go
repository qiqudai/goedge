//go:build !windows
// +build !windows

package main

import "syscall"

type signal = syscall.Signal

const (
	signalTerm = syscall.SIGTERM
	signalHup  = syscall.SIGHUP
)

func sendSignal(pid int, sig signal) error {
	return syscall.Kill(pid, sig)
}
