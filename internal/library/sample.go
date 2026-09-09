// Package library discovers and describes audio samples on disk.
package library

import "time"

// Format identifies the audio container/codec of a sample.
type Format string

const (
	FormatWAV  Format = "wav"
	FormatMP3  Format = "mp3"
	FormatFLAC Format = "flac"
	FormatOGG  Format = "ogg"
)

// Sample describes a single audio file discovered in a library, along with
// whatever metadata could reliably be extracted from it.
type Sample struct {
	Path   string
	Name   string
	Format Format
	Size   int64

	// The fields below are the zero value when metadata extraction is not
	// supported for the sample's format, or when extraction failed.
	Duration   time.Duration
	SampleRate int
	Channels   int
	BitDepth   int
}

// HasAudioMetadata reports whether decode-derived metadata (duration, sample
// rate, channels, bit depth) is present for this sample.
func (s Sample) HasAudioMetadata() bool {
	return s.SampleRate > 0
}
