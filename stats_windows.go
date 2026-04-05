//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

const processQueryInformation = 0x0400

var (
	psapiDLL                 = syscall.NewLazyDLL("psapi.dll")
	procGetProcessMemoryInfo = psapiDLL.NewProc("GetProcessMemoryInfo")
	kernel32DLL              = syscall.NewLazyDLL("kernel32.dll")
	procGetProcessIoCounters = kernel32DLL.NewProc("GetProcessIoCounters")
)

type processMemoryCounters struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type WindowsStats struct {
	ProcessStats
	PageFaults     uint64
	PeakWorkingSet uint64
	ReadOps        uint64
	WriteOps       uint64
	ReadBytes      uint64
	WriteBytes     uint64
}

func acquireHandle(cmd *exec.Cmd) ProcessHandle {
	if cmd.Process == nil {
		return ProcessHandle{}
	}

	handle, err := syscall.OpenProcess(processQueryInformation, false, uint32(cmd.Process.Pid))
	if err != nil {
		return ProcessHandle{}
	}

	return ProcessHandle{h: uintptr(handle)}
}

func releaseHandle(h ProcessHandle) {
	if h.h != 0 {
		syscall.CloseHandle(syscall.Handle(h.h))
	}
}

func collectIOBeforeWait(_ int) ioResult {
	return ioResult{}
}

func collectStats(h ProcessHandle, ps *os.ProcessState, setup, execTime, real time.Duration) WindowsStats {
	stats := WindowsStats{
		ProcessStats: ProcessStats{
			Real: real, Setup: setup, Exec: execTime,
			User: ps.UserTime(), Sys: ps.SystemTime(),
		},
	}

	if h.h == 0 {
		return stats
	}

	handle := syscall.Handle(h.h)

	var (
		cTime syscall.Filetime
		eTime syscall.Filetime
		kTime syscall.Filetime
		uTime syscall.Filetime
	)

	err := syscall.GetProcessTimes(handle, &cTime, &eTime, &kTime, &uTime)
	if err == nil {
		c := cTime.Nanoseconds()
		e := eTime.Nanoseconds()

		if c > 0 && e > c {
			stats.Exec = time.Duration(e - c)
		}
	}

	var pmc processMemoryCounters

	pmc.CB = uint32(unsafe.Sizeof(pmc))

	ret, _, _ := procGetProcessMemoryInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&pmc)), uintptr(pmc.CB))
	if ret != 0 {
		stats.PageFaults = uint64(pmc.PageFaultCount)
		stats.PeakWorkingSet = uint64(pmc.PeakWorkingSetSize)
	}

	var ioc ioCounters

	ret, _, _ = procGetProcessIoCounters.Call(uintptr(handle), uintptr(unsafe.Pointer(&ioc)))
	if ret != 0 {
		stats.ReadOps = ioc.ReadOperationCount
		stats.WriteOps = ioc.WriteOperationCount
		stats.ReadBytes = ioc.ReadTransferCount
		stats.WriteBytes = ioc.WriteTransferCount
	}

	return stats
}

func (s WindowsStats) entries() []StatEntry {
	e := []StatEntry{
		{"real", formatTime(s.Real), "Wall-clock time from start to finish", GroupTime},
		{"setup", formatTime(s.Setup), "Time to create process and set up pipes", GroupTime},
		{"exec", formatTime(s.Exec), "Time from process start until it exited", GroupTime},
		{"user", formatTime(s.User), "CPU time in user mode (code + libraries)", GroupTime},
		{"sys", formatTime(s.Sys), "CPU time in kernel mode (syscalls, page faults)", GroupTime},
	}

	if s.PeakWorkingSet > 0 {
		e = append(e, StatEntry{"maxrss", formatBytes(s.PeakWorkingSet), "Peak physical memory usage", GroupMemory})
	}

	if s.PageFaults > 0 {
		e = append(e, StatEntry{"pageflt", fmt.Sprintf("%d", s.PageFaults), "Total page faults", GroupMemory})
	}

	if s.ReadOps > 0 {
		e = append(e, StatEntry{"reads", fmt.Sprintf("%d (%s)", s.ReadOps, formatBytes(s.ReadBytes)), "Read operations and bytes transferred", GroupIO})
	}

	if s.WriteOps > 0 {
		e = append(e, StatEntry{"writes", fmt.Sprintf("%d (%s)", s.WriteOps, formatBytes(s.WriteBytes)), "Write operations and bytes transferred", GroupIO})
	}

	return e
}
