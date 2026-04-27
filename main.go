package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, out io.Writer, errOut io.Writer) int {
	// Spec: running the installed CLI with no args should monitor docker events.
	if len(args) < 2 {
		return monitorCmd(out, errOut)
	}

	switch args[1] {
	case "help", "-h", "--help":
		usage(out)
		return 0
	case "list":
		return listCmd(args[2:], out, errOut)
	default:
		fmt.Fprintf(errOut, "unknown command: %s\n\n", args[1])
		usage(errOut)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "lim")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  lim            # monitor docker events and persist last-run timestamps")
	fmt.Fprintln(w, "  lim list       # print images and their last-run timestamps")
	fmt.Fprintln(w, "  lim list --before Nh  # only show images last run more than Nh ago (hours only, e.g. 24h)")
	fmt.Fprintln(w, "  lim help")
	fmt.Fprintln(w)
}
