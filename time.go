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
	var (
		version bool
		explain bool
		full    bool
	)

	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			args = args[i+1:]

			break
		}

		if strings.HasPrefix(arg, "--") {
			switch arg {
			case "--version":
				version = true
			case "--explain":
				explain = true
			case "--full":
				full = true
			default:
				break
			}

			args = append(args[:i], args[i+1:]...)

			i--

			continue
		}

		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			var consumed bool

			for _, c := range arg[1:] {
				switch c {
				case 'v':
					version = true
					consumed = true
				case 'e':
					explain = true
					consumed = true
				case 'f':
					full = true
					consumed = true
				}
			}

			if consumed {
				args = append(args[:i], args[i+1:]...)

				i--
			}
		}
	}

	if version {
		fmt.Printf("time %s\n", Version)

		os.Exit(0)
	}

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: time [-f|--full] [-e|--explain] [-v|--version] <command> [args...]")

		os.Exit(1)
	}

	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	totalStart := time.Now()

	setupStart := time.Now()
	startErr := cmd.Start()
	setup := time.Since(setupStart)

	if startErr != nil {
		fmt.Fprintf(os.Stderr, "time: failed to start command: %v\n", startErr)

		os.Exit(1)
	}

	handle := acquireHandle(cmd)
	defer releaseHandle(handle)

	execStart := time.Now()
	waitErr := cmd.Wait()
	execTime := time.Since(execStart)

	real := time.Since(totalStart)

	stats := collectStats(handle, cmd.ProcessState, setup, execTime, real)

	printStats(stats.entries(), explain, full)

	if waitErr == nil {
		os.Exit(0)
	}

	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		os.Exit(exitCode(exitErr))
	}

	fmt.Fprintf(os.Stderr, "time: failed to run command: %v\n", waitErr)

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
