// Package format holds small presentation helpers shared by the CLI and TUI.
package format

import (
	"fmt"
	"time"
)

// HumanSize renders a byte count as a human-readable string (e.g. "144.7 KiB").
func HumanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// OptionalDuration renders d, or "-" when known is false (metadata not
// available for the sample's format).
func OptionalDuration(d time.Duration, known bool) string {
	if !known {
		return "-"
	}
	return d.Round(time.Millisecond).String()
}

// OptionalInt renders v with suffix appended, or "-" when known is false.
func OptionalInt(v int, known bool, suffix string) string {
	if !known {
		return "-"
	}
	return fmt.Sprintf("%d%s", v, suffix)
}
