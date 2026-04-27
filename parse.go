package main

import (
	"strings"
	"time"
)

// parseDockerEventsLine extracts (image, timestamp) from a single line of `docker events` output.
// We treat `container create` events as the closest proxy to `docker run`.
func parseDockerEventsLine(line string) (image string, ts time.Time, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", time.Time{}, false
	}

	// Docker events begins with an RFC3339Nano timestamp.
	space := strings.IndexByte(line, ' ')
	if space <= 0 {
		return "", time.Time{}, false
	}
	tsStr := line[:space]
	parsedTs, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		return "", time.Time{}, false
	}

	// Example:
	// 2026-04-27T07:29:50.123456789Z container create <id> (image=alpine, name=foo)
	if !strings.Contains(line, " container create ") {
		return "", time.Time{}, false
	}

	idx := strings.Index(line, "image=")
	if idx < 0 {
		return "", time.Time{}, false
	}
	start := idx + len("image=")
	end := start
	for end < len(line) {
		switch line[end] {
		case ',', ')':
			goto done
		default:
			end++
		}
	}

done:
	img := strings.TrimSpace(line[start:end])
	if img == "" {
		return "", time.Time{}, false
	}

	return img, parsedTs, true
}
