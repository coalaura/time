//go:build !linux && !windows

package main

import (
	"os"
	"os/exec"
	"time"
)

func acquireHandle(cmd *exec.Cmd) ProcessHandle {
	return ProcessHandle{}
}

func releaseHandle(_ ProcessHandle) {}

func collectIOBeforeWait(_ int) ioResult {
	return ioResult{}
}

func collectStats(_ ProcessHandle, ps *os.ProcessState, setup, execTime, real time.Duration) ProcessStats {
	return ProcessStats{
		Real: real, Setup: setup, Exec: execTime,
		User: ps.UserTime(), Sys: ps.SystemTime(),
	}
}

func (s ProcessStats) entries() []StatEntry {
	return []StatEntry{
		{"real", formatTime(s.Real), "Wall-clock time from start to finish", GroupTime},
		{"setup", formatTime(s.Setup), "Time to create process and set up pipes", GroupTime},
		{"exec", formatTime(s.Exec), "Time from process start until it exited", GroupTime},
		{"user", formatTime(s.User), "CPU time in user mode (code + libraries)", GroupTime},
		{"sys", formatTime(s.Sys), "CPU time in kernel mode (syscalls, page faults)", GroupTime},
	}
}
