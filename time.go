package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: time <command> [args...]")

		os.Exit(1)
	}

	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--version" {
			fmt.Printf("time %s\n", Version)

			os.Exit(0)
		}
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()

	err := cmd.Run()

	real := time.Since(start)

	user, sys := getCPUTime(cmd.ProcessState)

	fmt.Fprintf(os.Stderr, "real\t%s\n", formatTime(real))
	fmt.Fprintf(os.Stderr, "user\t%s\n", formatTime(user))
	fmt.Fprintf(os.Stderr, "sys\t%s\n", formatTime(sys))

	if err == nil {
		os.Exit(0)
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitCode(exitErr))
	}

	fmt.Fprintf(os.Stderr, "time: failed to run command: %v\n", err)

	os.Exit(1)
}

func formatTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	if d == 0 {
		return "0s"
	}

	type unit struct {
		value  time.Duration
		suffix string
	}

	units := []unit{
		{time.Hour, "h"},
		{time.Minute, "m"},
		{time.Second, "s"},
		{time.Millisecond, "ms"},
		{time.Microsecond, "µs"},
	}

	parts := make([]string, 0, len(units))

	for _, unit := range units {
		if d < unit.value {
			continue
		}

		count := d / unit.value
		d -= count * unit.value

		parts = append(parts, fmt.Sprintf("%d%s", count, unit.suffix))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%dns", d/time.Nanosecond)
	}

	return strings.Join(parts, " ")
}

func exitCode(err *exec.ExitError) int {
	if err == nil {
		return 0
	}

	if status, ok := err.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus()
	}

	return 1
}

func getCPUTime(ps *os.ProcessState) (user, sys time.Duration) {
	if ps == nil {
		return 0, 0
	}

	user = ps.UserTime()
	sys = ps.SystemTime()

	return user, sys
}
