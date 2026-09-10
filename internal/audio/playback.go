package audio

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
)

// playbackSampleRate is the fixed rate the speaker is opened at; streams
// decoded at a different native rate are resampled to it.
const playbackSampleRate beep.SampleRate = 44100

var (
	speakerOnce sync.Once
	speakerErr  error
)

func initSpeaker() error {
	speakerOnce.Do(func() {
		bufferSize := playbackSampleRate.N(time.Second / 10)
		speakerErr = speaker.Init(playbackSampleRate, bufferSize)
	})
	return speakerErr
}

// playback tracks a single in-flight Play call so it can be stopped or
// cleaned up exactly once, whether that happens because it finished
// naturally or because something else asked it to stop.
type playback struct {
	streamer beep.StreamSeekCloser
	done     chan struct{}
	once     sync.Once
}

func (pb *playback) finish() {
	pb.once.Do(func() {
		pb.streamer.Close()
		close(pb.done)
	})
}

// Player plays one sample at a time asynchronously through the system's
// audio output. The zero value is not usable; construct with NewPlayer.
type Player struct {
	mu      sync.Mutex
	current *playback
}

// NewPlayer creates a Player. The audio device is not opened until the
// first call to Play.
func NewPlayer() *Player {
	return &Player{}
}

// Play decodes and starts playing the sample at path (whose container is
// identified by sampleFormat, e.g. "wav"), stopping whatever the player was
// previously playing. The returned channel is closed when playback finishes
// naturally or Stop is called.
func (p *Player) Play(path string, sampleFormat string) (<-chan struct{}, error) {
	if err := initSpeaker(); err != nil {
		return nil, fmt.Errorf("audio: initializing speaker: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var (
		streamer     beep.StreamSeekCloser
		streamFormat beep.Format
	)
	switch sampleFormat {
	case "wav":
		streamer, streamFormat, err = wav.Decode(f)
	case "mp3":
		streamer, streamFormat, err = mp3.Decode(f)
	case "flac":
		streamer, streamFormat, err = flac.Decode(f)
	case "ogg":
		streamer, streamFormat, err = vorbis.Decode(f)
	default:
		f.Close()
		return nil, fmt.Errorf("audio: playback not supported for format %q", sampleFormat)
	}
	if err != nil {
		return nil, fmt.Errorf("audio: decoding %s: %w", path, err)
	}

	pb := &playback{streamer: streamer, done: make(chan struct{})}
	resampled := beep.Resample(4, streamFormat.SampleRate, playbackSampleRate, streamer)

	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()

	speaker.Play(beep.Seq(resampled, beep.Callback(func() {
		p.mu.Lock()
		if p.current == pb {
			p.current = nil
		}
		p.mu.Unlock()
		pb.finish()
	})))
	p.current = pb

	return pb.done, nil
}

// Stop halts playback, if any is in progress.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
}

// stopLocked requires p.mu to be held.
func (p *Player) stopLocked() {
	if p.current == nil {
		return
	}
	speaker.Clear()
	pb := p.current
	p.current = nil
	pb.finish()
}
