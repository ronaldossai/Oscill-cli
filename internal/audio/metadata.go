// Package audio extracts metadata from audio files.
//
// For the MVP, only WAV is decoded well enough to yield reliable duration,
// sample rate, channel count and bit depth — its header is simple enough to
// parse with the standard library alone. Compressed formats (MP3, FLAC, OGG)
// are not decoded yet; callers still get filename/size/format from the
// library scanner, and this package returns ErrUnsupportedFormat for them
// rather than guessing.
package audio

import (
	"errors"
	"time"
)

// ErrUnsupportedFormat is returned by Probe when metadata extraction for a
// format is not yet implemented.
var ErrUnsupportedFormat = errors.New("audio: metadata extraction not supported for this format")

// Metadata holds audio properties derived from decoding a file's header.
type Metadata struct {
	Duration   time.Duration
	SampleRate int
	Channels   int
	BitDepth   int
}

// Probe extracts metadata for the file at path, whose container/codec is
// identified by format (e.g. "wav", "mp3", "flac", "ogg").
func Probe(path string, format string) (Metadata, error) {
	switch format {
	case "wav":
		return probeWAV(path)
	default:
		return Metadata{}, ErrUnsupportedFormat
	}
}
