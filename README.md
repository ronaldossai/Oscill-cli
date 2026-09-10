# Oscill

A terminal toolkit for music producers to browse, analyse, and manage audio
sample libraries — built in Go.

Oscill is not a DAW. It's a fast, Unix-style tool for working with sample
libraries from the terminal.

## Status

Phases 1–3 (see [Roadmap](#roadmap)): recursive sample discovery and metadata
extraction (`scan`), an interactive TUI browser (`oscill` / `browse`), and
sample playback/preview (`space`), plus a MIDI drum-pad mode. Search and
audio analysis are not built yet.

## Getting Started

Browse a library interactively:

```sh
go run ./cmd/oscill ~/Music/Samples
# or: go run ./cmd/oscill browse ~/Music/Samples
```

Navigate with `↑`/`↓` (or `j`/`k`), `tab` to switch between the folders and
samples panes, `enter` to open a folder, `backspace` to go up, `space` to
preview/stop the selected sample, `q` to quit. Playback is decoded and
played asynchronously (via [gopxl/beep](https://github.com/gopxl/beep)), so
the UI stays responsive while a sample plays. Supported for preview: WAV,
MP3, FLAC, OGG.

### MIDI drum pads

Plug in a MIDI controller with pads (tested with an Arturia MiniLab mkII)
and press `m` to connect to it. Oscill doesn't assume any particular pad
layout — it *learns* your controller by listening for taps: tap each pad
once, pause for a couple of seconds, and it locks in however many distinct
pads it saw, in the order you tapped them. Press `m` again at any time to
relearn (e.g. if you switch pad banks on the controller).

To load a sample onto a pad, highlight it in the Samples pane and press `a`,
then tap the pad you want it on — the status bar walks you through it. Once
assigned, tapping that pad plays the sample directly, no keyboard needed,
with the Pads panel flashing to confirm the hit. Notes that aren't one of
the learned pads (e.g. the controller's keybed) are ignored.

This uses [gitlab.com/gomidi/midi/v2](https://gitlab.com/gomidi/midi) with
its `rtmididrv` driver, which requires cgo. On macOS this just needs the
Xcode Command Line Tools (`xcode-select --install`) — RtMidi's C++ source is
vendored in the driver and links against the system CoreMIDI framework, no
Homebrew packages required. Linux builds will additionally need ALSA
development headers (e.g. `libasound2-dev` on Debian/Ubuntu) available at
build time; this hasn't been tested on Linux yet.

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
                   and playback (playback.go)
internal/format/   Shared display formatting (sizes, durations)
internal/midi/     MIDI input: listing ports and receiving note events
internal/ui/       The interactive TUI browser (Bubble Tea), including the
                   drum-pad mode (pads.go)
```

## Roadmap

1. **Scan** — recursive discovery + metadata (done)
2. **Interactive TUI browser** (done)
3. **Sample playback/preview** (done)
4. Search & filtering
5. Audio analysis (BPM, key, RMS, spectrum)
6. Persistent index/cache

Beyond the original plan: **MIDI drum-pad mode** (done) — connect a MIDI
pad controller, auto-detect its pad count, and assign/trigger samples live.
