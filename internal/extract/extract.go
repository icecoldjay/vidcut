package extract

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/icecoldjay/vidcut/internal/timecode"
)

// Options configures a single video extract operation.
type Options struct {
	Input     string
	Output    string
	Start     time.Duration
	End       time.Duration
	Duration  time.Duration
	HasEnd    bool
	HasDur    bool
	Accurate  bool // frame-accurate re-encode (slower)
	Overwrite bool
	DryRun    bool
	Verbose   bool
	FFmpeg    string // defaults to "ffmpeg"

	// MediaDuration is set by Validate after probing.
	MediaDuration time.Duration
}

// ClipDuration returns the length of the clip to extract.
func (o *Options) ClipDuration() time.Duration {
	if o.HasDur {
		return o.Duration
	}
	return o.End - o.Start
}

// Validate checks paths and time ranges before invoking ffmpeg.
func (o *Options) Validate() error {
	if strings.TrimSpace(o.Input) == "" {
		return fmt.Errorf("input video path is required")
	}
	info, err := os.Stat(o.Input)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", o.Input)
		}
		return fmt.Errorf("cannot read input file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("input path is a directory, not a file: %s", o.Input)
	}
	if info.Size() == 0 {
		return fmt.Errorf("input file is empty: %s", o.Input)
	}

	if o.Start < 0 {
		return fmt.Errorf("start time must be non-negative")
	}

	if o.HasEnd && o.HasDur {
		return fmt.Errorf("specify either --end or --duration, not both")
	}
	if !o.HasEnd && !o.HasDur {
		return fmt.Errorf("one of --end or --duration is required")
	}

	clipDur := o.ClipDuration()
	if o.HasEnd {
		if o.End <= o.Start {
			return fmt.Errorf("end (%s) must be after start (%s)",
				timecode.FormatFFmpeg(o.End), timecode.FormatFFmpeg(o.Start))
		}
	} else if clipDur <= 0 {
		return fmt.Errorf("duration must be greater than zero")
	}

	if strings.TrimSpace(o.Output) == "" {
		return fmt.Errorf("output path is required")
	}
	outDir := filepath.Dir(o.Output)
	if outDir == "" {
		outDir = "."
	}
	dirInfo, err := os.Stat(outDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("output directory does not exist: %s", outDir)
		}
		return fmt.Errorf("cannot access output directory: %w", err)
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("output parent path is not a directory: %s", outDir)
	}

	absIn, err := filepath.Abs(o.Input)
	if err != nil {
		return fmt.Errorf("resolve input path: %w", err)
	}
	absOut, err := filepath.Abs(o.Output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if absIn == absOut {
		return fmt.Errorf("output path must differ from input path")
	}

	if !o.Overwrite {
		if _, err := os.Stat(o.Output); err == nil {
			return fmt.Errorf("output already exists (use --overwrite): %s", o.Output)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("cannot stat output path: %w", err)
		}
	}

	bin := o.ffmpegBin()
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH (install ffmpeg, or set --ffmpeg): %w", err)
	}

	mediaDur, err := probeDuration(o.ffprobeBin(), o.Input)
	if err != nil {
		return fmt.Errorf("could not read media duration (is this a video file?): %w", err)
	}
	o.MediaDuration = mediaDur

	if o.Start >= mediaDur {
		return fmt.Errorf("start (%s) is at or beyond media duration (%s)",
			timecode.FormatFFmpeg(o.Start), timecode.FormatFFmpeg(mediaDur))
	}
	if o.HasEnd && o.End > mediaDur {
		return fmt.Errorf("end (%s) is beyond media duration (%s)",
			timecode.FormatFFmpeg(o.End), timecode.FormatFFmpeg(mediaDur))
	}
	if o.Start+clipDur > mediaDur {
		return fmt.Errorf("clip (start %s + duration %s) exceeds media duration (%s)",
			timecode.FormatFFmpeg(o.Start), timecode.FormatFFmpeg(clipDur), timecode.FormatFFmpeg(mediaDur))
	}

	return nil
}

func (o *Options) ffmpegBin() string {
	if o.FFmpeg != "" {
		return o.FFmpeg
	}
	return "ffmpeg"
}

func (o *Options) ffprobeBin() string {
	bin := o.ffmpegBin()
	if bin == "ffmpeg" {
		return "ffprobe"
	}
	dir := filepath.Dir(bin)
	base := filepath.Base(bin)
	probe := strings.Replace(base, "ffmpeg", "ffprobe", 1)
	if dir == "." || dir == "" {
		return probe
	}
	return filepath.Join(dir, probe)
}

func probeDuration(ffprobe, input string) (time.Duration, error) {
	cmd := exec.Command(ffprobe,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		input,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration %q: %w", strings.TrimSpace(string(out)), err)
	}
	if sec <= 0 {
		return 0, fmt.Errorf("invalid media duration %v", sec)
	}
	return time.Duration(sec * float64(time.Second)), nil
}

// Args builds the ffmpeg argument list (excluding the binary name).
func (o *Options) Args() []string {
	start := timecode.FormatFFmpeg(o.Start)
	dur := timecode.FormatFFmpeg(o.ClipDuration())

	args := []string{"-hide_banner"}
	if o.Verbose {
		args = append(args, "-loglevel", "info", "-stats")
	} else {
		args = append(args, "-loglevel", "error")
	}

	if o.Overwrite {
		args = append(args, "-y")
	} else {
		args = append(args, "-n")
	}

	if o.Accurate {
		// Seek after opening input for frame-accurate cuts, then re-encode.
		args = append(args,
			"-i", o.Input,
			"-ss", start,
			"-t", dur,
			"-map", "0:v:0",
			"-map", "0:a?",
			"-map_metadata", "0",
			"-c:v", "libx264",
			"-preset", "fast",
			"-crf", "18",
			"-c:a", "aac",
		)
	} else {
		// Fast path: input seek + stream copy (keeps video/audio/subs/metadata).
		args = append(args,
			"-ss", start,
			"-i", o.Input,
			"-t", dur,
			"-map", "0",
			"-map_metadata", "0",
			"-c", "copy",
		)
	}

	if isMP4Family(o.Output) {
		args = append(args, "-movflags", "+faststart")
	}

	args = append(args, o.Output)
	return args
}

func isMP4Family(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".m4v", ".mov":
		return true
	default:
		return false
	}
}

// CommandLine returns a shell-ish preview of the ffmpeg invocation.
func (o *Options) CommandLine() string {
	parts := []string{shellQuote(o.ffmpegBin())}
	for _, a := range o.Args() {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return `''`
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Run extracts the requested portion of the input video.
func (o *Options) Run() error {
	if err := o.Validate(); err != nil {
		return err
	}

	if o.Verbose || o.DryRun {
		fmt.Fprintf(os.Stderr, "media duration: %s\n", timecode.FormatFFmpeg(o.MediaDuration))
		fmt.Fprintf(os.Stderr, "clip: %s → +%s\n",
			timecode.FormatFFmpeg(o.Start), timecode.FormatFFmpeg(o.ClipDuration()))
		mode := "stream copy (keyframe-aligned)"
		if o.Accurate {
			mode = "accurate re-encode"
		}
		fmt.Fprintf(os.Stderr, "mode: %s\n", mode)
		fmt.Fprintf(os.Stderr, "cmd: %s\n", o.CommandLine())
	}

	if o.DryRun {
		return nil
	}

	cmd := exec.Command(o.ffmpegBin(), o.Args()...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		hint := ""
		if !o.Accurate {
			hint = " (try --accurate if stream copy failed due to codecs/subtitles)"
		}
		return fmt.Errorf("ffmpeg failed%s: %w", hint, err)
	}

	info, err := os.Stat(o.Output)
	if err != nil {
		return fmt.Errorf("output was not created: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("output file is empty: %s", o.Output)
	}
	return nil
}

// DefaultOutput builds <name>_clip<ext> next to the input when -o is omitted.
func DefaultOutput(input string) string {
	dir := filepath.Dir(input)
	base := filepath.Base(input)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if ext == "" {
		ext = ".mp4"
	}
	return filepath.Join(dir, name+"_clip"+ext)
}
