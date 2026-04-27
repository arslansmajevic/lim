package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func rootCmd(args []string, out io.Writer, errOut io.Writer) int {
	fs := flag.NewFlagSet("lim", flag.ContinueOnError)
	fs.SetOutput(errOut)
	statusOnly := fs.Bool("status", false, "Print status and exit")
	shutdown := fs.Bool("shutdown", false, "Stop the background docker-events monitor")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *statusOnly {
		if err := printStatus(out); err != nil {
			fmt.Fprintf(errOut, "status error: %v\n", err)
			return 1
		}
		return 0
	}

	if *shutdown {
		if err := shutdownMonitor(); err != nil {
			fmt.Fprintf(errOut, "shutdown error: %v\n", err)
			printUsageAndStatus(out)
			return 1
		}
		printUsageAndStatus(out)
		return 0
	}

	// Default UX: show usage + whether we're connected, and ensure the monitor is running.
	if err := ensureMonitorRunning(errOut); err != nil {
		fmt.Fprintf(errOut, "%v\n", err)
		printUsageAndStatus(out)
		return 1
	}

	printUsageAndStatus(out)
	return 0
}

func printUsageAndStatus(out io.Writer) {
	usage(out)
	fmt.Fprintln(out)
	_ = printStatusLines(out)
}

func printStatus(out io.Writer) error {
	return printStatusLines(out)
}

func printStatusLines(out io.Writer) error {
	fmt.Fprintln(out, "Status:")

	if err := dockerAvailableCheck(); err != nil {
		fmt.Fprintf(out, "  docker: unavailable (%v)\n", err)
	} else {
		fmt.Fprintln(out, "  docker: available")
	}

	running, heartbeatAge, ok := monitorHealth(nowProvider())
	if !running {
		fmt.Fprintln(out, "  monitor: not running")
		return nil
	}
	if ok {
		fmt.Fprintf(out, "  monitor: running (heartbeat %s ago)\n", heartbeatAge.Truncate(time.Second))
		return nil
	}
	fmt.Fprintln(out, "  monitor: running (status unknown)")
	return nil
}

func ensureMonitorRunning(errOut io.Writer) error {
	if err := dockerAvailableCheck(); err != nil {
		return fmt.Errorf("docker unavailable: %w", err)
	}

	now := nowProvider().UTC()
	running, age, ok := monitorHealth(now)
	if running {
		// If we can detect staleness, attempt a restart.
		if ok && age > 30*time.Second {
			fmt.Fprintln(errOut, "lim: monitor seems unhealthy; restarting")
			_ = shutdownMonitor()
			return startMonitorBackground()
		}
		return nil
	}

	return startMonitorBackground()
}

func monitorHealth(now time.Time) (running bool, heartbeatAge time.Duration, hasHeartbeat bool) {
	running = isMonitorRunning()
	if !running {
		return false, 0, false
	}

	st, ok, err := readMonitorStatus()
	if err != nil || !ok || st.LastHeartbeat.IsZero() {
		return true, 0, false
	}

	if st.LastHeartbeat.After(now) {
		return true, 0, true
	}
	return true, now.Sub(st.LastHeartbeat), true
}

func isMonitorRunning() bool {
	lockPath, err := monitorLockPath()
	if err != nil {
		return false
	}
	lock, acquired, err := acquireLock(lockPath)
	if err != nil {
		return false
	}
	if acquired {
		_ = lock.release()
		return false
	}
	return true
}

func monitorLockPath() (string, error) {
	stateFile, err := stateFilePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(stateFile), "monitor.lock"), nil
}

func startMonitorBackground() error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	cmd := exec.Command(self, "_monitor")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	applyDaemonAttrs(cmd.SysProcAttr)

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err == nil {
		defer devNull.Close()
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start monitor: %w", err)
	}

	return nil
}

func shutdownMonitor() error {
	lockPath, err := monitorLockPath()
	if err != nil {
		return err
	}

	pid, ok := monitorPIDFromStatusOrLock(lockPath)
	if !ok {
		// Nothing running.
		return nil
	}

	if err := terminateProcess(pid); err != nil {
		return err
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !isMonitorRunning() {
			_ = clearMonitorStatus()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return errors.New("monitor did not stop in time")
}

func monitorPIDFromStatusOrLock(lockPath string) (int, bool) {
	st, ok, err := readMonitorStatus()
	if err == nil && ok && st.PID > 0 {
		return st.PID, true
	}

	b, err := os.ReadFile(lockPath)
	if err != nil {
		return 0, false
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return 0, false
	}
	pid, err := strconv.Atoi(s)
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}
