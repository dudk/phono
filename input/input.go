// Package input provides types to parse user input of pipe components.
package input

import (
	"io"

	"github.com/pipelined/mp3"
	"github.com/pipelined/pipe"
	"github.com/pipelined/wav"
)

type (
	// Pump contains all pipe.Pumps that user can provide as input.
	Pump struct {
		Wav *wav.Pump
		Mp3 *mp3.Pump
	}

	// Sink contains all pipe.Sinks that user can provide as input.
	Sink struct {
		Mp3 *mp3.Sink
		Wav *wav.Sink
	}
)

func (s Sink) mp3() bool {
	return s.Mp3 != nil
}

func (s Sink) wav() bool {
	return s.Wav != nil
}

func (p Pump) mp3() bool {
	return p.Mp3 != nil
}

func (p Pump) wav() bool {
	return p.Wav != nil
}

// SetOutput to the sink provided as input.
func (s Sink) SetOutput(ws io.WriteSeeker) {
	switch {
	case s.mp3():
		s.Mp3.Writer = ws
	case s.wav():
		s.Wav.WriteSeeker = ws
	}
}

// Extension of the file for the sink.
func (s Sink) Extension() string {
	switch {
	case s.mp3():
		return mp3.DefaultExtension
	case s.wav():
		return wav.DefaultExtension
	default:
		return ""
	}
}

// Sink provided as input.
func (s Sink) Sink() pipe.Sink {
	switch {
	case s.mp3():
		return s.Mp3
	case s.wav():
		return s.Wav
	default:
		return nil
	}
}

// Pump provided as input.
func (p Pump) Pump() pipe.Pump {
	switch {
	case p.mp3():
		return p.Mp3
	case p.wav():
		return p.Wav
	default:
		return nil
	}
}
