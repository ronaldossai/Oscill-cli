# Oscill

A terminal toolkit for music producers to browse, analyse, and manage audio
sample libraries — built in Go.

Oscill is not a DAW. It's a fast, Unix-style tool for working with sample
libraries from the terminal.

## Status

Phases 1–4 (see [Roadmap](#roadmap)): recursive sample discovery and metadata
extraction (`scan`), an interactive TUI browser (`oscill` / `browse`), sample
playback/preview (`space`), and search & filtering (`find`, `/`), plus a MIDI
drum-pad mode. Audio analysis (BPM/key/etc.) is not built yet.

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

### Search & filtering

Inside the browser, press `/` to search the whole library live — not just
the current folder. Type to filter by name or folder as you go (e.g. "kick"
matches both `kick_808.wav` and anything under a `Kicks/` folder); results
show their path so you can tell same-named files in different folders
apart. Press `enter` to stop editing the query and navigate the results
normally (`space` still previews, `a` still assigns to a pad); press `enter`
again on a result to jump to its folder, or `esc` at any point to clear the
search and go back to normal browsing.

From the command line, `oscill find [directory] [flags]` does the same
recursive search non-interactively:

```sh
oscill find ~/Music/Samples --name kick
oscill find ~/Music/Samples --format wav --max-duration 2s
```

Flags (any order relative to the directory): `--name` (substring match on
filename or path), `--format` (wav/mp3/flac/ogg), `--min-duration` /
`--max-duration` (Go duration strings like `500ms`, `1.5s`). Duration
filters only match samples with known duration (currently WAV).

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

Beside the Folders panel, a MIDI telemetry panel shows every note event as
it arrives — timestamp, on/off, note number, velocity, and which pad (if
any) it maps to — regardless of whether it's currently doing anything, so
you can watch exactly what your controller is sending, including notes that
aren't mapped to a pad (e.g. to see what the keybed sends before deciding
whether to map it).

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
internal/library/  Sample discovery (scanner, list.go), the Sample type,
                   and search/filtering (search.go)
internal/audio/    Per-format metadata extraction (decoder.go, metadata.go)
                   and playback (playback.go)
internal/format/   Shared display formatting (sizes, durations)
internal/midi/     MIDI input: listing ports and receiving note events
internal/ui/       The interactive TUI browser (Bubble Tea), including the
                   drum-pad mode (pads.go), MIDI telemetry (telemetry.go),
                   and live search (search.go)
```

## Roadmap

1. **Scan** — recursive discovery + metadata (done)
2. **Interactive TUI browser** (done)
3. **Sample playback/preview** (done)
4. **Search & filtering** (done)
5. Audio analysis (BPM, key, RMS, spectrum)
6. Persistent index/cache

Beyond the original plan: **MIDI drum-pad mode** (done) — connect a MIDI
pad controller, auto-detect its pad count, and assign/trigger samples live.
