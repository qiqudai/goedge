//go:build windows
// +build windows

package main

import "errors"

type signal int

const (
	signalTerm signal = 15
	signalHup  signal = 1
)

func sendSignal(pid int, sig signal) error {
	return errors.New("signal is not supported on windows")
}
