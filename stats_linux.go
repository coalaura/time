//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	_WNOWAIT = 0x01000000
	_WEXITED = 0x00000004
	_P_PID   = 1
)

type linuxIOStats struct {
	rchar      uint64
	wchar      uint64
	syscr      uint64
	syscw      uint64
	readBytes  uint64
	writeBytes uint64
}

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

func collectIOBeforeWait(pid int) ioResult {
	var info [128]byte

	_, _, err := syscall.Syscall6(
		syscall.SYS_WAITID,
		uintptr(_P_PID),
		uintptr(pid),
		uintptr(unsafe.Pointer(&info[0])),
		uintptr(_WEXITED|_WNOWAIT),
		0,
		0,
	)

	if err != 0 {
		return ioResult{}
	}

	return ioResult{
		data:     readProcIO(pid),
		exitedAt: time.Now(),
	}
}

func readProcIO(pid int) *linuxIOStats {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/io", pid))
	if err != nil {
		return nil
	}

	stats := &linuxIOStats{}

	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		val, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil {
			continue
		}

		switch strings.TrimSpace(parts[0]) {
		case "rchar":
			stats.rchar = val
		case "wchar":
			stats.wchar = val
		case "syscr":
			stats.syscr = val
		case "syscw":
			stats.syscw = val
		case "read_bytes":
			stats.readBytes = val
		case "write_bytes":
			stats.writeBytes = val
		}
	}

	return stats
}

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
		{"user", formatTime(s.User), "CPU time in user mode (your code + libraries)", GroupTime},
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

func (s LinuxStats) ioEntries(data any) []StatEntry {
	io, ok := data.(*linuxIOStats)
	if !ok || io == nil {
		return nil
	}

	var e []StatEntry

	if io.syscr > 0 {
		e = append(e, StatEntry{"syscr", fmt.Sprintf("%d", io.syscr), "Read syscalls", GroupIO})
	}

	if io.syscw > 0 {
		e = append(e, StatEntry{"syscw", fmt.Sprintf("%d", io.syscw), "Write syscalls", GroupIO})
	}

	if io.readBytes > 0 {
		e = append(e, StatEntry{"read", formatBytes(io.readBytes), "Bytes read from storage (not cache)", GroupIO})
	}

	if io.writeBytes > 0 {
		e = append(e, StatEntry{"write", formatBytes(io.writeBytes), "Bytes written to storage", GroupIO})
	}

	if io.rchar > 0 {
		e = append(e, StatEntry{"rchar", formatBytes(io.rchar), "Bytes read (including from cache)", GroupIO})
	}

	if io.wchar > 0 {
		e = append(e, StatEntry{"wchar", formatBytes(io.wchar), "Bytes written", GroupIO})
	}

	return e
}
