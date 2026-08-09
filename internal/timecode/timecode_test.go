package timecode

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"90", 90 * time.Second},
		{"90s", 90 * time.Second},
		{"90.5", 90*time.Second + 500*time.Millisecond},
		{"1:30", 90 * time.Second},
		{"01:02:03", time.Hour + 2*time.Minute + 3*time.Second},
		{"1:20:30", time.Hour + 20*time.Minute + 30*time.Second},
		{"1:20:30.25", time.Hour + 20*time.Minute + 30250*time.Millisecond},
		{"12:05:00", 12*time.Hour + 5*time.Minute},
		{"1:02:03.5", time.Hour + 2*time.Minute + 3500*time.Millisecond},
		{"1m30s", 90 * time.Second},
		{"2m", 2 * time.Minute},
		{"1h2m3s", time.Hour + 2*time.Minute + 3*time.Second},
		{"1h20m30s", time.Hour + 20*time.Minute + 30*time.Second},
		{"0", 0},
	}

	for _, tc := range tests {
		got, err := Parse(tc.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("Parse(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	for _, in := range []string{"", "abc", "1:60", "1:2:60", "-5", "90:30"} {
		if _, err := Parse(in); err == nil {
			t.Fatalf("Parse(%q) expected error", in)
		}
	}
}

func TestFormatFFmpeg(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{time.Hour + 2*time.Minute + 3*time.Second + 250*time.Millisecond, "1:02:03.250"},
		{time.Hour + 20*time.Minute + 30*time.Second, "1:20:30.000"},
		{12*time.Hour + 5*time.Minute, "12:05:00.000"},
	}
	for _, tc := range tests {
		got := FormatFFmpeg(tc.in)
		if got != tc.want {
			t.Fatalf("FormatFFmpeg(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
