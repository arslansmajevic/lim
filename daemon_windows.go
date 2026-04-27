//go:build windows

package main

import "syscall"

func applyDaemonAttrs(cmdSysProcAttr *syscall.SysProcAttr) {
	// No-op.
}
