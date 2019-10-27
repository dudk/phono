package file

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/pipelined/flac"
	"github.com/pipelined/mp3"
	"github.com/pipelined/pipe"
	"github.com/pipelined/signal"
	"github.com/pipelined/wav"
)

type (
	wavFormat struct {
		DefaultExtension string
		Extensions       map[string]struct{}
		BitDepths        map[signal.BitDepth]struct{}
	}

	mp3Format struct {
		DefaultExtension string
		Extensions       map[string]struct{}
		MaxBitRate       int
		MinBitRate       int
		ChannelModes     map[mp3.ChannelMode]struct{}
		VBR              string
		CBR              string
		ABR              string
	}

	flacFormat struct {
		DefaultExtension string
		Extensions       map[string]struct{}
	}

	// BuildPumpFunc is used to inject ReadSeeker into Pump.
	BuildPumpFunc func(io.ReadSeeker) pipe.Pump

	// BuildSinkFunc is used to inject WriteSeeker into Sink.
	BuildSinkFunc func(io.WriteSeeker) pipe.Sink
)

// Maybe use this to replace BuildSink?
// type OutputOptions interface {
// 	Sink() pipe.Sink
// }

var (
	// Wav provides structures required to handle wav files.
	Wav = wavFormat{
		DefaultExtension: ".wav",
		Extensions: map[string]struct{}{
			".wav":  {},
			".wave": {},
		},
		BitDepths: map[signal.BitDepth]struct{}{
			signal.BitDepth8:  {},
			signal.BitDepth16: {},
			signal.BitDepth24: {},
			signal.BitDepth32: {},
		},
	}

	// Mp3 provides structures required to handle mp3 files.
	Mp3 = mp3Format{
		DefaultExtension: ".mp3",
		Extensions: map[string]struct{}{
			".mp3": {},
		},
		MinBitRate: 8,
		MaxBitRate: 320,
		ChannelModes: map[mp3.ChannelMode]struct{}{
			mp3.JointStereo: {},
			mp3.Stereo:      {},
			mp3.Mono:        {},
		},
		VBR: "VBR",
		ABR: "ABR",
		CBR: "CBR",
	}

	// Flac provides structures required to handle flac files.
	Flac = flacFormat{
		DefaultExtension: ".flac",
		Extensions: map[string]struct{}{
			".flac": {},
		},
	}
)

// Pump returns pump for provided file source. Type of the pump is determined by file extension.
func Pump(fileName string) (BuildPumpFunc, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch {
	case hasExtension(ext, Wav.Extensions):
		return func(rs io.ReadSeeker) pipe.Pump {
			return &wav.Pump{ReadSeeker: rs}
		}, nil
	case hasExtension(ext, Mp3.Extensions):
		return func(rs io.ReadSeeker) pipe.Pump {
			return &mp3.Pump{Reader: rs}
		}, nil
	case hasExtension(ext, Flac.Extensions):
		return func(rs io.ReadSeeker) pipe.Pump {
			return &flac.Pump{Reader: rs}
		}, nil
	default:
		return nil, fmt.Errorf("File has unsupported extension: %v", fileName)
	}
}

// hasExtension validates if filename has one of passed extensions.
// Filename is lower-cased before comparison.
func hasExtension(ext string, exts map[string]struct{}) bool {
	_, ok := exts[ext]
	return ok
}

// BuildSink validates all parameters required to build wav sink. If valid, build closure is returned.
// Closure allows to postpone io opertaions and do them only after all sink parameters are validated.
func (f wavFormat) BuildSink(bitDepth int) (BuildSinkFunc, error) {
	bd := signal.BitDepth(bitDepth)
	if _, ok := f.BitDepths[bd]; !ok {
		return nil, fmt.Errorf("Bit depth %v is not supported", bitDepth)
	}

	return func(ws io.WriteSeeker) pipe.Sink {
		return &wav.Sink{
			BitDepth:    bd,
			WriteSeeker: ws,
		}
	}, nil
}

// BuildSink validates all parameters required to build mp3 sink. If valid, build closure is returned.
// Closure allows to postpone io opertaions and do them only after all sink parameters are validated.
func (f mp3Format) BuildSink(bitRateMode string, bitRate, channelMode int, useQuality bool, quality int) (BuildSinkFunc, error) {
	cm := mp3.ChannelMode(channelMode)
	if _, ok := f.ChannelModes[cm]; !ok {
		return nil, fmt.Errorf("Channel mode %v is not supported", cm)
	}

	var brm mp3.BitRateMode
	switch strings.ToUpper(bitRateMode) {
	case f.VBR:
		if bitRate < 0 || bitRate > 9 {
			return nil, fmt.Errorf("VBR quality %v is not supported", bitRate)
		}
		brm = mp3.VBR(bitRate)
	case f.CBR:
		if err := f.bitRate(bitRate); err != nil {
			return nil, err
		}
		brm = mp3.CBR(bitRate)
	case f.ABR:
		if err := f.bitRate(bitRate); err != nil {
			return nil, err
		}
		brm = mp3.ABR(bitRate)
	default:
		return nil, fmt.Errorf("Bit rate mode %v is not supported", bitRateMode)
	}

	if useQuality {
		if quality < 0 || quality > 9 {
			return nil, fmt.Errorf("MP3 quality %v is not supported", quality)
		}
	}

	return func(ws io.WriteSeeker) pipe.Sink {
		s := &mp3.Sink{
			BitRateMode: brm,
			ChannelMode: cm,
			Writer:      ws,
		}
		if useQuality {
			s.SetQuality(quality)
		}
		return s
	}, nil
}

// BitRate checks if provided bit rate is supported.
func (f mp3Format) bitRate(v int) error {
	if v > f.MaxBitRate || v < f.MinBitRate {
		return fmt.Errorf("Bit rate %v is not supported. Provide value between %d and %d", v, f.MinBitRate, f.MaxBitRate)
	}
	return nil
}
