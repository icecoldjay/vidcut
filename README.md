# vidcut

CLI utility that extracts a portion of a local video file. Validates paths and
time ranges, then cuts with [FFmpeg](https://ffmpeg.org/).

## Requirements

- [FFmpeg](https://ffmpeg.org/) (and `ffprobe`) on your `PATH`
  - macOS: `brew install ffmpeg`
  - Ubuntu/Debian: `sudo apt install ffmpeg`

## Install

### From source

```bash
go install github.com/icecoldjay/vidcut@latest
```

### From a release binary

Download the archive for your OS from [GitHub Releases](https://github.com/icecoldjay/vidcut/releases), unpack, and put `vidcut` on your `PATH`.

### Homebrew (after you publish a tap)

```bash
brew install icecoldjay/tap/vidcut
```

### Build locally

```bash
git clone https://github.com/icecoldjay/vidcut.git
cd vidcut
go build -o vidcut .
```

## Usage

```bash
# Long video: start at 1h 20m 30s, end at 1h 25m
vidcut -i lecture.mp4 -s 1:20:30 -e 1:25:00 -o clip.mp4

# Same window via duration
vidcut -i lecture.mp4 -s 1:20:30 -d 4m30s -o clip.mp4

# Compound times
vidcut -i lecture.mp4 -s 1h20m30s -d 5m -o clip.mp4

# Frame-accurate cut (re-encodes; slower)
vidcut -i input.mp4 -s 1:20:30 -d 30s --accurate -o clip.mp4

# Preview the plan without writing a file
vidcut -i input.mp4 -s 1:20:30 -d 30s --dry-run -v

# Overwrite an existing output
vidcut -i input.mp4 -s 10s -d 5s -o clip.mp4 --overwrite
```

Positional input is also supported:

```bash
vidcut lecture.mp4 -s 1:20:30 -d 2m
# → writes lecture_clip.mp4 next to the source
```

### Time formats

| Example | Meaning |
|---------|---------|
| `90` / `90s` | 90 seconds |
| `1:30` | 1 minute 30 seconds |
| `1:20:30` | **1 hour 20 minutes 30 seconds** (use this for long media) |
| `12:05:00.5` | 12 hours 5 minutes 0.5 seconds |
| `1h20m30s` | same as `1:20:30` |

`MM:SS` minutes must be `< 60`. For anything past an hour, use `H:MM:SS` or
`NhNmNs` — not `80:30`.

### Accuracy

| Mode | Flag | Behavior |
|------|------|----------|
| Fast (default) | _(none)_ | Stream copy; seek is keyframe-aligned (may start slightly early/late) |
| Accurate | `--accurate` / `--reencode` | Re-encodes with post-input seek for frame-accurate cuts |

If stream copy fails (unusual codecs/subtitles), retry with `--accurate`.

### Flags

| Flag | Description |
|------|-------------|
| `-i`, `--input` | Input video path |
| `-o`, `--output` | Output path (default: `<name>_clip.<ext>`) |
| `-s`, `--start` | Start time (default `0`) |
| `-e`, `--end` | End time (exclusive with `--duration`) |
| `-d`, `--duration` | Clip length (exclusive with `--end`) |
| `--accurate` | Frame-accurate re-encode |
| `--overwrite`, `-y` | Allow overwriting an existing output |
| `--dry-run` | Print plan / ffmpeg command only |
| `-v`, `--verbose` | Show duration, mode, and command |
| `--ffmpeg` | Path to ffmpeg binary |
| `--version` | Print build version |

## Publish a release (GoReleaser)

```bash
git tag v0.1.0
git push origin main --tags
```

The [release workflow](.github/workflows/release.yml) runs GoReleaser and
attaches binaries + checksums to the [GitHub Release](https://github.com/icecoldjay/vidcut/releases).

Optional Homebrew: create `icecoldjay/homebrew-tap`, uncomment the `brews:`
block in `.goreleaser.yaml`, and add a `HOMEBREW_TAP_GITHUB_TOKEN` secret if
the tap is a separate repo.

Local snapshot build (no publish):

```bash
goreleaser release --snapshot --clean
```

## License

MIT
