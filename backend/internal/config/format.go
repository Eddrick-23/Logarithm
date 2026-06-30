package config

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func formatDuration(d time.Duration) string {
	// custom duration formatter since package conversion function returns 1h0m0s instead of 1h
	if d == 0 {
		return "0s"
	}

	var b strings.Builder

	days := int64(d / (24 * time.Hour))
	d -= time.Duration(days) * 24 * time.Hour

	hours := int64(d / time.Hour)
	d -= time.Duration(hours) * time.Hour

	minutes := int64(d / time.Minute)
	d -= time.Duration(minutes) * time.Minute

	seconds := d.Seconds() // remaining, may be fractional

	if days > 0 {
		fmt.Fprintf(&b, "%dd ", days)
	}
	if hours > 0 {
		fmt.Fprintf(&b, "%dh ", hours)
	}
	if minutes > 0 {
		fmt.Fprintf(&b, "%dmin ", minutes)
	}
	if seconds > 0 {
		if seconds == math.Trunc(seconds) {
			fmt.Fprintf(&b, "%ds", int64(seconds))
		} else {
			fmt.Fprintf(&b, "%gs", seconds)
		}
	}

	return b.String()
}

func formatBackoff(durations []time.Duration) string {
	parts := make([]string, len(durations))
	for i, d := range durations {
		parts[i] = formatDuration(d)
	}
	return strings.Join(parts, ", ")
}
