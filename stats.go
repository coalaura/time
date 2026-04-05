package main

import (
	"fmt"
	"os"
	"time"
)

const (
	GroupTime = iota
	GroupMemory
	GroupIO
	GroupSwitches

	KB = 1024
	MB = KB * 1024
	GB = MB * 1024
)

type ProcessStats struct {
	Real  time.Duration
	Setup time.Duration
	Exec  time.Duration
	User  time.Duration
	Sys   time.Duration
}

type ProcessHandle struct {
	h uintptr
}

type StatEntry struct {
	Key     string
	Value   string
	Explain string
	Group   int
}

type ioResult struct {
	data     any
	exitedAt time.Time
}

type statsWithIO interface {
	entries() []StatEntry
	ioEntries(any) []StatEntry
}

func formatBytes(b uint64) string {
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.0fKB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func printStats(entries []StatEntry, ioEntries []StatEntry, explain bool, full bool) {
	if !full {
		var (
			real string
			user string
			sys  string
		)

		for _, e := range entries {
			switch e.Key {
			case "real":
				real = e.Value
			case "user":
				user = e.Value
			case "sys":
				sys = e.Value
			}
		}

		fmt.Fprintf(os.Stderr, "real\t%s\n", real)
		fmt.Fprintf(os.Stderr, "user\t%s\n", user)
		fmt.Fprintf(os.Stderr, "sys\t%s\n", sys)

		return
	}

	all := append(entries, ioEntries...)

	lastGroup := -1

	for _, e := range all {
		if e.Group != lastGroup && lastGroup != -1 {
			fmt.Fprintln(os.Stderr)
		}

		fmt.Fprintf(os.Stderr, "%s\t%s\n", e.Key, e.Value)

		if explain {
			fmt.Fprintf(os.Stderr, "  \x1b[90;3m%s\x1b[0m\n", e.Explain)
		}

		lastGroup = e.Group
	}
}
