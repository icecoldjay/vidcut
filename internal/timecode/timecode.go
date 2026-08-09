package timecode

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// H:MM:SS, HH:MM:SS, or MM:SS (minutes must be < 60 when hours are present;
	// without hours, MM:SS still requires minutes < 60 — use 90 or 1:30:00 for longer).
	hmsRe = regexp.MustCompile(`^(?:(\d+):)?(\d{1,2}):(\d{1,2}(?:\.\d+)?)$`)
	durRe = regexp.MustCompile(`(?i)^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+(?:\.\d+)?)s)?$`)
)

// Parse converts flexible time strings into a duration from 00:00:00.
// Accepted forms:
//   - bare seconds: "90", "90.5"
//   - suffix seconds: "90s"
//   - compound: "1h2m3s", "2m", "1m30s", "1h20m30s"
//   - clock MM:SS: "1:30", "01:30"
//   - clock H:MM:SS (long videos): "1:20:30", "12:05:00.5"
func Parse(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty time value")
	}

	if d, err := parseClock(s); err == nil {
		return d, nil
	}

	if d, err := parseCompound(s); err == nil {
		return d, nil
	}

	if d, err := parseSeconds(s); err == nil {
		return d, nil
	}

	return 0, fmt.Errorf("invalid time %q (try 90, 1:30, 1:20:30, 1h20m30s)", s)
}

func parseSeconds(s string) (time.Duration, error) {
	s = strings.TrimSuffix(strings.ToLower(s), "s")
	sec, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if sec < 0 {
		return 0, fmt.Errorf("time must be non-negative")
	}
	return time.Duration(sec * float64(time.Second)), nil
}

func parseClock(s string) (time.Duration, error) {
	m := hmsRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("not clock format")
	}

	var hours, minutes int
	var seconds float64
	var err error

	if m[1] != "" {
		hours, err = strconv.Atoi(m[1])
		if err != nil {
			return 0, err
		}
	}
	minutes, err = strconv.Atoi(m[2])
	if err != nil {
		return 0, err
	}
	seconds, err = strconv.ParseFloat(m[3], 64)
	if err != nil {
		return 0, err
	}

	if minutes >= 60 {
		return 0, fmt.Errorf("minutes out of range in %q (use H:MM:SS for long times, e.g. 1:20:30)", s)
	}
	if seconds >= 60 {
		return 0, fmt.Errorf("seconds out of range in %q", s)
	}

	d := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds*float64(time.Second))
	return d, nil
}

func parseCompound(s string) (time.Duration, error) {
	lower := strings.ToLower(s)
	if !strings.ContainsAny(lower, "hms") {
		return 0, fmt.Errorf("not compound format")
	}
	if durRe.FindString(lower) != lower {
		return 0, fmt.Errorf("not compound format")
	}
	m := durRe.FindStringSubmatch(lower)
	if m == nil || (m[1] == "" && m[2] == "" && m[3] == "") {
		return 0, fmt.Errorf("not compound format")
	}

	var d time.Duration
	if m[1] != "" {
		h, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, err
		}
		d += time.Duration(h) * time.Hour
	}
	if m[2] != "" {
		min, err := strconv.Atoi(m[2])
		if err != nil {
			return 0, err
		}
		d += time.Duration(min) * time.Minute
	}
	if m[3] != "" {
		sec, err := strconv.ParseFloat(m[3], 64)
		if err != nil {
			return 0, err
		}
		d += time.Duration(sec * float64(time.Second))
	}
	return d, nil
}

// FormatFFmpeg returns an H:MM:SS.mmm string suitable for ffmpeg -ss/-t/-to.
// Hours are unpadded beyond 2 digits so multi-hour media formats cleanly (e.g. 12:05:00.000).
func FormatFFmpeg(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalMs := d.Milliseconds()
	hours := totalMs / 3_600_000
	totalMs %= 3_600_000
	minutes := totalMs / 60_000
	totalMs %= 60_000
	seconds := totalMs / 1000
	ms := totalMs % 1000
	return fmt.Sprintf("%d:%02d:%02d.%03d", hours, minutes, seconds, ms)
}
