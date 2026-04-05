//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type LinuxStats struct {
	ProcessStats
	MaxRSS         uint64
	MinorFaults    uint64
	MajorFaults    uint64
	InputBlocks    uint64
	OutputBlocks   uint64
	VoluntaryCSW   uint64
	InvoluntaryCSW uint64
}

func acquireHandle(cmd *exec.Cmd) ProcessHandle {
	return ProcessHandle{}
}

func releaseHandle(_ ProcessHandle) {}

func collectStats(_ ProcessHandle, ps *os.ProcessState, setup, execTime, real time.Duration) LinuxStats {
	stats := LinuxStats{
		ProcessStats: ProcessStats{
			Real: real, Setup: setup, Exec: execTime,
			User: ps.UserTime(), Sys: ps.SystemTime(),
		},
	}

	var ru syscall.Rusage

	err := syscall.Getrusage(syscall.RUSAGE_CHILDREN, &ru)
	if err == nil {
		stats.MaxRSS = uint64(ru.Maxrss) * 1024
		stats.MinorFaults = uint64(ru.Minflt)
		stats.MajorFaults = uint64(ru.Majflt)
		stats.InputBlocks = uint64(ru.Inblock)
		stats.OutputBlocks = uint64(ru.Oublock)
		stats.VoluntaryCSW = uint64(ru.Nvcsw)
		stats.InvoluntaryCSW = uint64(ru.Nivcsw)
	}

	return stats
}

func (s LinuxStats) entries() []StatEntry {
	e := []StatEntry{
		{"real", formatTime(s.Real), "Wall-clock time from start to finish", GroupTime},
		{"setup", formatTime(s.Setup), "Time to fork, exec, and set up pipes", GroupTime},
		{"exec", formatTime(s.Exec), "Time from process start until it exited", GroupTime},
		{"user", formatTime(s.User), "CPU time in user mode (code + libraries)", GroupTime},
		{"sys", formatTime(s.Sys), "CPU time in kernel mode (syscalls, page faults)", GroupTime},
	}

	if s.MaxRSS > 0 {
		e = append(e, StatEntry{"maxrss", formatBytes(s.MaxRSS), "Peak resident set size (physical memory)", GroupMemory})
	}

	if s.MinorFaults > 0 {
		e = append(e, StatEntry{"minflt", fmt.Sprintf("%d", s.MinorFaults), "Minor page faults (no disk I/O)", GroupMemory})
	}

	if s.MajorFaults > 0 {
		e = append(e, StatEntry{"majflt", fmt.Sprintf("%d", s.MajorFaults), "Major page faults (required disk read)", GroupMemory})
	}

	if s.InputBlocks > 0 {
		e = append(e, StatEntry{"inblock", fmt.Sprintf("%d", s.InputBlocks), "Block reads from filesystem (512-byte units)", GroupIO})
	}

	if s.OutputBlocks > 0 {
		e = append(e, StatEntry{"oublock", fmt.Sprintf("%d", s.OutputBlocks), "Block writes to filesystem (512-byte units)", GroupIO})
	}

	if s.VoluntaryCSW > 0 {
		e = append(e, StatEntry{"nvcsw", fmt.Sprintf("%d", s.VoluntaryCSW), "Voluntary context switches (yielded CPU)", GroupSwitches})
	}

	if s.InvoluntaryCSW > 0 {
		e = append(e, StatEntry{"nivcsw", fmt.Sprintf("%d", s.InvoluntaryCSW), "Involuntary context switches (preempted)", GroupSwitches})
	}

	return e
}
