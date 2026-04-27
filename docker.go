package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var dockerAvailableCheck = checkDockerAvailable

func checkDockerAvailable() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker CLI not found in PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("docker check timed out")
	}
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("docker not available: %s", msg)
	}

	if strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("docker daemon not reachable")
	}

	return nil
}
