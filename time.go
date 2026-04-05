package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
	execStart := time.Now()
	setup := execStart.Sub(setupStart)

	if startErr != nil {
		fmt.Fprintf(os.Stderr, "time: failed to start command: %v\n", startErr)

		os.Exit(1)
	}

	handle := acquireHandle(cmd)
	defer releaseHandle(handle)

	ioResult := collectIOBeforeWait(cmd.Process.Pid)
	waitErr := cmd.Wait()

	var execTime time.Duration

	if !ioResult.exitedAt.IsZero() {
		execTime = ioResult.exitedAt.Sub(execStart)
	} else {
		execTime = time.Since(execStart)
	}

	real := time.Since(totalStart)

	stats := collectStats(handle, cmd.ProcessState, setup, execTime, real)

	var ioEntries []StatEntry

	if s, ok := any(stats).(statsWithIO); ok {
		ioEntries = s.ioEntries(ioResult.data)
	}

	printStats(stats.entries(), ioEntries, explain, full)

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

	var b strings.Builder

	b.Grow(16)

	for _, unit := range units {
		if d < unit.value {
			continue
		}

		count := d / unit.value
		d -= count * unit.value

		if b.Len() > 0 {
			b.WriteByte(' ')
		}

		b.WriteString(strconv.FormatInt(int64(count), 10))
		b.WriteString(unit.suffix)
	}

	if b.Len() == 0 {
		b.WriteString(strconv.FormatInt(int64(d/time.Nanosecond), 10))
		b.WriteString("ns")
	}

	return b.String()
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
