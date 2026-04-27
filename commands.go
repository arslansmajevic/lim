package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var nowProvider = time.Now

func monitorCmd(out io.Writer, errOut io.Writer) int {
	if err := checkDockerAvailable(); err != nil {
		fmt.Fprintf(errOut, "docker unavailable: %v\n", err)
		return 1
	}

	stateFile, err := stateFilePath()
	if err != nil {
		fmt.Fprintf(errOut, "state path error: %v\n", err)
		return 1
	}

	lockPath := filepath.Join(filepath.Dir(stateFile), "monitor.lock")
	lock, acquired, err := acquireLock(lockPath)
	if err != nil {
		fmt.Fprintf(errOut, "monitor lock error: %v\n", err)
		return 1
	}
	if !acquired {
		fmt.Fprintln(out, "lim: monitor already running")
		return 0
	}
	defer func() { _ = lock.release() }()

	st, err := loadImageState(stateFile)
	if err != nil {
		fmt.Fprintf(errOut, "state load error: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var cmd *exec.Cmd
	var stdout io.ReadCloser
	var stderr io.ReadCloser
	for attempt := 1; attempt <= 3; attempt++ {
		cmd = exec.CommandContext(ctx, "docker",
			"events",
			"--filter", "type=container",
			"--filter", "event=create",
		)

		stdout, err = cmd.StdoutPipe()
		if err != nil {
			fmt.Fprintf(errOut, "docker events stdout: %v\n", err)
			return 1
		}
		stderr, err = cmd.StderrPipe()
		if err != nil {
			fmt.Fprintf(errOut, "docker events stderr: %v\n", err)
			return 1
		}

		if err := cmd.Start(); err == nil {
			break
		} else if attempt == 3 {
			fmt.Fprintf(errOut, "start docker events: %v\n", err)
			return 1
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	// Forward stderr in the background.
	go func() {
		_, _ = io.Copy(errOut, stderr)
	}()

	scanner := bufio.NewScanner(io.TeeReader(stdout, out))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		image, ts, ok := parseDockerEventsLine(line)
		if !ok {
			continue
		}
		// Persist the newest timestamp per image.
		if err := st.setLastRun(image, ts); err != nil {
			fmt.Fprintf(errOut, "state save error (%s): %v\n", image, err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(errOut, "read docker events: %v\n", err)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return 1
	}

	err = cmd.Wait()
	if err == nil || ctx.Err() == context.Canceled {
		return 0
	}
	fmt.Fprintf(errOut, "docker events exited: %v\n", err)
	return 1
}

func listCmd(args []string, out io.Writer, errOut io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(errOut)
	before := fs.String("before", "", "Only show images last run more than Nh ago (hours only, e.g. 24h)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	stateFile, err := stateFilePath()
	if err != nil {
		fmt.Fprintf(errOut, "state path error: %v\n", err)
		return 1
	}
	st, err := loadImageState(stateFile)
	if err != nil {
		fmt.Fprintf(errOut, "state load error: %v\n", err)
		return 1
	}

	keys := make([]string, 0, len(st.images))
	for k := range st.images {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var cutoff time.Time
	if *before != "" {
		dur, err := parseBeforeHours(*before)
		if err != nil {
			fmt.Fprintf(errOut, "invalid --before value %q: %v\n", *before, err)
			return 2
		}
		cutoff = nowProvider().UTC().Add(-dur)
	}

	for _, image := range keys {
		ts := st.images[image]
		if !cutoff.IsZero() && !ts.Before(cutoff) {
			continue
		}
		fmt.Fprintf(out, "%s\t%s\n", image, ts.UTC().Format(time.RFC3339Nano))
	}

	return 0
}

func parseBeforeHours(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if !strings.HasSuffix(s, "h") {
		return 0, fmt.Errorf("must be in hours with 'h' suffix (e.g. 24h)")
	}
	nStr := strings.TrimSuffix(s, "h")
	if nStr == "" {
		return 0, fmt.Errorf("missing hours value")
	}
	n, err := strconv.Atoi(nStr)
	if err != nil {
		return 0, fmt.Errorf("hours must be an integer")
	}
	if n < 0 {
		return 0, fmt.Errorf("hours must be >= 0")
	}
	return time.Duration(n) * time.Hour, nil
}
