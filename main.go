package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/icecoldjay/vidcut/internal/extract"
	"github.com/icecoldjay/vidcut/internal/timecode"
)

// Set by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("vidcut", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		input     string
		output    string
		startStr  string
		endStr    string
		durStr    string
		accurate  bool
		reencode  bool // alias for --accurate
		overwrite bool
		dryRun    bool
		verbose   bool
		showVer   bool
		ffmpeg    string
	)

	fs.StringVar(&input, "i", "", "input video path")
	fs.StringVar(&input, "input", "", "input video path")
	fs.StringVar(&output, "o", "", "output video path (default: <input>_clip.<ext>)")
	fs.StringVar(&output, "output", "", "output video path")
	fs.StringVar(&startStr, "s", "0", "start time (e.g. 1:20:30, 90, 1h20m30s)")
	fs.StringVar(&startStr, "start", "0", "start time")
	fs.StringVar(&endStr, "e", "", "end time (exclusive with --duration)")
	fs.StringVar(&endStr, "end", "", "end time")
	fs.StringVar(&durStr, "d", "", "clip duration (exclusive with --end)")
	fs.StringVar(&durStr, "duration", "", "clip duration")
	fs.BoolVar(&accurate, "accurate", false, "frame-accurate cut via re-encode (slower)")
	fs.BoolVar(&reencode, "reencode", false, "alias for --accurate")
	fs.BoolVar(&overwrite, "overwrite", false, "overwrite output if it already exists")
	fs.BoolVar(&overwrite, "y", false, "alias for --overwrite")
	fs.BoolVar(&dryRun, "dry-run", false, "print the ffmpeg plan without writing output")
	fs.BoolVar(&verbose, "verbose", false, "show media info, mode, and ffmpeg command")
	fs.BoolVar(&verbose, "v", false, "alias for --verbose")
	fs.BoolVar(&showVer, "version", false, "print version and exit")
	fs.StringVar(&ffmpeg, "ffmpeg", "ffmpeg", "path to ffmpeg binary")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `vidcut — extract a portion of a local video file

Usage:
  vidcut -i lecture.mp4 -s 1:20:30 -e 1:25:00 -o clip.mp4
  vidcut -i input.mp4 -s 80s -d 90s -o clip.mp4
  vidcut input.mp4 -s 1h20m -d 5m --accurate

Time formats (including multi-hour media):
  90 | 90s | 1:30 | 1:20:30 | 1h20m30s | 12:05:00.5

Flags:
`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	if showVer {
		fmt.Printf("vidcut %s (%s/%s) commit=%s built=%s\n",
			version, runtime.GOOS, runtime.GOARCH, commit, date)
		return nil
	}

	if input == "" {
		switch fs.NArg() {
		case 0:
			fs.Usage()
			return fmt.Errorf("input video path is required")
		case 1:
			input = fs.Arg(0)
		default:
			return fmt.Errorf("unexpected arguments: %v", fs.Args())
		}
	} else if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}

	start, err := timecode.Parse(startStr)
	if err != nil {
		return fmt.Errorf("start: %w", err)
	}

	opts := extract.Options{
		Input:     input,
		Output:    output,
		Start:     start,
		Accurate:  accurate || reencode,
		Overwrite: overwrite,
		DryRun:    dryRun,
		Verbose:   verbose,
		FFmpeg:    ffmpeg,
	}

	if endStr != "" {
		end, err := timecode.Parse(endStr)
		if err != nil {
			return fmt.Errorf("end: %w", err)
		}
		opts.End = end
		opts.HasEnd = true
	}
	if durStr != "" {
		dur, err := timecode.Parse(durStr)
		if err != nil {
			return fmt.Errorf("duration: %w", err)
		}
		opts.Duration = dur
		opts.HasDur = true
	}

	if opts.Output == "" {
		opts.Output = extract.DefaultOutput(opts.Input)
	}

	if err := opts.Run(); err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Println("dry-run OK (no file written)")
		return nil
	}

	fmt.Printf("wrote %s (%s → +%s)\n",
		opts.Output,
		timecode.FormatFFmpeg(opts.Start),
		timecode.FormatFFmpeg(opts.ClipDuration()),
	)
	return nil
}
