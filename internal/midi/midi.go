// Package midi provides minimal MIDI input support: listing connected
// controllers and receiving note on/off events from one of them.
package midi

import (
	"fmt"

	gomidi "gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // registers the system MIDI driver
)

// NoteEvent is a note on/off message from a MIDI input port.
type NoteEvent struct {
	Channel  uint8
	Key      uint8
	Velocity uint8
	On       bool
}

// Ports lists the names of currently available MIDI input ports.
func Ports() ([]string, error) {
	ins := gomidi.GetInPorts()
	names := make([]string, len(ins))
	for i, in := range ins {
		names[i] = in.String()
	}
	return names, nil
}

// Listen opens the named MIDI input port and delivers note on/off events on
// the returned channel until the returned close function is called. The
// channel is never closed by Listen itself (only stops receiving); callers
// should stop consuming it once they call close.
func Listen(portName string) (<-chan NoteEvent, func(), error) {
	in, err := gomidi.FindInPort(portName)
	if err != nil {
		return nil, nil, fmt.Errorf("midi: port %q not found: %w", portName, err)
	}

	events := make(chan NoteEvent, 32)
	stop, err := gomidi.ListenTo(in, func(msg gomidi.Message, _ int32) {
		var ch, key, vel uint8
		var ev NoteEvent
		switch {
		case msg.GetNoteOn(&ch, &key, &vel):
			ev = NoteEvent{Channel: ch, Key: key, Velocity: vel, On: vel > 0}
		case msg.GetNoteOff(&ch, &key, &vel):
			ev = NoteEvent{Channel: ch, Key: key, Velocity: vel, On: false}
		default:
			return
		}
		// Never block the driver's callback: drop the event if the reader
		// is behind rather than stalling MIDI input.
		select {
		case events <- ev:
		default:
		}
	})
	if err != nil {
		return nil, nil, fmt.Errorf("midi: listening to %q: %w", portName, err)
	}

	return events, stop, nil
}
