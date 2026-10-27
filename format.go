package portaudio

import (
	"fmt"
	"strings"
)

// SampleFormat is a sample format (PaSampleFormat).
type SampleFormat uint32

const (
	Float32 SampleFormat = 0x00000001
	Int32   SampleFormat = 0x00000002
	// Int24 is packed 24-bit samples. Use the Raw stream methods to read or write it.
	Int24        SampleFormat = 0x00000004
	Int16        SampleFormat = 0x00000008
	Int8         SampleFormat = 0x00000010
	UInt8        SampleFormat = 0x00000020
	CustomFormat SampleFormat = 0x00010000
	// NonInterleaved can be ORed with a format to use one buffer per channel.
	NonInterleaved SampleFormat = 0x80000000
)

var sampleFormatNames = []struct {
	format SampleFormat
	name   string
}{
	{Float32, "Float32"},
	{Int32, "Int32"},
	{Int24, "Int24"},
	{Int16, "Int16"},
	{Int8, "Int8"},
	{UInt8, "UInt8"},
	{CustomFormat, "CustomFormat"},
	{NonInterleaved, "NonInterleaved"},
}

func (f SampleFormat) String() string {
	var names []string
	rest := f
	for _, n := range sampleFormatNames {
		if f&n.format != 0 {
			names = append(names, n.name)
			rest &^= n.format
		}
	}
	if rest != 0 || len(names) == 0 {
		names = append(names, fmt.Sprintf("0x%x", uint32(rest)))
	}
	return strings.Join(names, "|")
}

// Sample is the set of Go types that map directly to a PortAudio sample format.
type Sample interface {
	float32 | int32 | int16 | int8 | uint8
}

func sampleFormatOf[T Sample]() SampleFormat {
	var zero T
	switch any(zero).(type) {
	case float32:
		return Float32
	case int32:
		return Int32
	case int16:
		return Int16
	case int8:
		return Int8
	default:
		return UInt8
	}
}

// StreamFlags modify the behavior of a stream (PaStreamFlags).
type StreamFlags uint32

const (
	NoFlag    StreamFlags = 0
	ClipOff   StreamFlags = 0x00000001
	DitherOff StreamFlags = 0x00000002
	// NeverDropInput is only valid for full duplex callback streams opened with
	// FramesPerBufferUnspecified.
	NeverDropInput                        StreamFlags = 0x00000004
	PrimeOutputBuffersUsingStreamCallback StreamFlags = 0x00000008
	PlatformSpecificFlags                 StreamFlags = 0xFFFF0000
)

// FramesPerBufferUnspecified lets the host choose the buffer size.
const FramesPerBufferUnspecified = 0

// StreamCallbackFlags report buffer problems to a stream callback
// (PaStreamCallbackFlags).
type StreamCallbackFlags uint32

const (
	InputUnderflow  StreamCallbackFlags = 0x00000001
	InputOverflow   StreamCallbackFlags = 0x00000002
	OutputUnderflow StreamCallbackFlags = 0x00000004
	OutputOverflow  StreamCallbackFlags = 0x00000008
	PrimingOutput   StreamCallbackFlags = 0x00000010
)

// StreamCallbackResult tells PortAudio whether to keep calling a stream callback.
type StreamCallbackResult int

const (
	// Continue keeps the stream running.
	Continue StreamCallbackResult = 0
	// Complete stops the stream after the pending output has been played.
	Complete StreamCallbackResult = 1
	// Abort stops the stream as soon as possible.
	Abort StreamCallbackResult = 2
)
