// internal/utils/timeago.go
package utils

import (
	"fmt"
	"time"
)

func FormatTimeAgoFromDiffSeconds(diffSec int64) string {
	if diffSec < 5 {
		return "just now"
	}
	if diffSec < 60 {
		return fmt.Sprintf("%d seconds ago", diffSec)
	}
	if diffSec < 3600 {
		return fmt.Sprintf("%d mins ago", diffSec/60)
	}
	return "long time ago"
}

func FormatTimeAgoFromTimestamp(tsMs int64) string {
	diffSec := (time.Now().UnixMilli() - tsMs) / 1000
	return FormatTimeAgoFromDiffSeconds(diffSec)
}

func FormatTimeAgoFromISO(iso string) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	diffSec := int64(time.Since(t).Seconds())
	return FormatTimeAgoFromDiffSeconds(diffSec)
}