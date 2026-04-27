//go:build !windows

package main

import "syscall"

func applyDaemonAttrs(cmdSysProcAttr *syscall.SysProcAttr) {
	// Detach from controlling terminal so the monitor can run in background.
	cmdSysProcAttr.Setsid = true
}
