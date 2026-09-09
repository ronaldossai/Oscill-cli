# Oscill

A terminal toolkit for music producers to browse, analyse, and manage audio
sample libraries — built in Go.

Oscill is not a DAW. It's a fast, Unix-style tool for working with sample
libraries from the terminal.

## Status

Phases 1–2 (see [Roadmap](#roadmap)): recursive sample discovery and metadata
extraction (`scan`), plus an interactive TUI browser (`oscill` / `browse`).
Playback, search, and audio analysis are not built yet.

## Getting Started

Browse a library interactively:

```sh
go run ./cmd/oscill ~/Music/Samples
# or: go run ./cmd/oscill browse ~/Music/Samples
```

Navigate with `↑`/`↓` (or `j`/`k`), `tab` to switch between the folders and
samples panes, `enter` to open a folder, `backspace` to go up, `q` to quit.

Print a one-shot metadata table instead:

```sh
go run ./cmd/oscill scan ~/Music/Samples
```

```text
NAME           FORMAT  SIZE       DURATION  SAMPLE RATE  CHANNELS  BIT DEPTH
kick_808.wav   wav     144.7 KiB  840ms     44100Hz      2         16-bit
kick_mono.wav  wav     140.7 KiB  1.5s      48000Hz      1         16-bit
loop.flac      flac    3.2 MiB    -         -             -        -

3 sample(s) found
```

Supported formats: WAV, MP3, FLAC, OGG (discovery for all four; full
duration/sample-rate/channel/bit-depth metadata is currently only decoded for
WAV — other formats show `-` rather than a guessed value).

Build a standalone binary:

```sh
go build -o oscill ./cmd/oscill
./oscill scan ~/Music/Samples
```

## Project Layout

```text
cmd/oscill/        CLI entry point
internal/library/  Sample discovery (scanner, list.go) and the Sample type
internal/audio/    Per-format metadata extraction (decoder.go, metadata.go)
internal/format/   Shared display formatting (sizes, durations)
internal/ui/       The interactive TUI browser (Bubble Tea)
```

## Roadmap

1. **Scan** — recursive discovery + metadata (done)
2. **Interactive TUI browser** (done)
3. Sample playback/preview
4. Search & filtering
5. Audio analysis (BPM, key, RMS, spectrum)
6. Persistent index/cache
