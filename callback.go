package portaudio

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// StreamCallback processes one buffer of audio. in is empty for output-only
// streams and out is empty for input-only streams; out must be filled completely.
//
// It runs on PortAudio's audio thread, so it should not block. The buffers are
// only valid until it returns.
type StreamCallback func(in, out Buffer, timeInfo StreamCallbackTimeInfo, flags StreamCallbackFlags) StreamCallbackResult

// StreamCallbackTimeInfo holds the timestamps of a callback's buffers, in seconds
// on the same clock as Stream.Time.
type StreamCallbackTimeInfo struct {
	InputBufferADCTime  float64
	CurrentTime         float64
	OutputBufferDACTime float64
}

// Buffer is audio memory passed to a StreamCallback. It is only valid during the
// callback.
type Buffer struct {
	ptr      unsafe.Pointer
	frames   int
	channels int
	format   SampleFormat
}

// Pointer returns the raw buffer. For NonInterleaved formats it points to an
// array of one pointer per channel.
func (b Buffer) Pointer() unsafe.Pointer { return b.ptr }

// Frames returns the number of frames in the buffer.
func (b Buffer) Frames() int { return b.frames }

// Channels returns the number of channels in the buffer.
func (b Buffer) Channels() int { return b.channels }

// Format returns the sample format of the buffer.
func (b Buffer) Format() SampleFormat { return b.format }

// Samples returns the interleaved samples of b, or nil if b is empty. It panics if
// T does not match b's sample format.
func Samples[T Sample](b Buffer) []T {
	if b.ptr == nil {
		return nil
	}
	b.mustBe(sampleFormatOf[T]())
	return unsafe.Slice((*T)(b.ptr), b.frames*b.channels)
}

// ChannelSamples returns the samples of one channel of a NonInterleaved buffer, or
// nil if b is empty. It panics if T does not match b's sample format or channel is
// out of range.
func ChannelSamples[T Sample](b Buffer, channel int) []T {
	if b.ptr == nil {
		return nil
	}
	b.mustBe(sampleFormatOf[T]() | NonInterleaved)
	if channel < 0 || channel >= b.channels {
		panic(fmt.Sprintf("portaudio: channel %d out of range [0, %d)", channel, b.channels))
	}
	channels := unsafe.Slice((*unsafe.Pointer)(b.ptr), b.channels)
	return unsafe.Slice((*T)(channels[channel]), b.frames)
}

func (b Buffer) mustBe(format SampleFormat) {
	if b.format != format {
		panic(fmt.Sprintf("portaudio: buffer format is %v, not %v", b.format, format))
	}
}

// purego callbacks can never be freed and their number is limited, so one
// trampoline per C callback type is shared by all streams.
var (
	callbacksOnce       sync.Once
	streamCallbackPtr   uintptr
	finishedCallbackPtr uintptr
)

func initCallbacks() {
	callbacksOnce.Do(func() {
		streamCallbackPtr = purego.NewCallback(streamCallbackTrampoline)
		finishedCallbackPtr = purego.NewCallback(finishedCallbackTrampoline)
	})
}

func streamCallbackAddr() uintptr {
	initCallbacks()
	return streamCallbackPtr
}

func finishedCallbackAddr() uintptr {
	initCallbacks()
	return finishedCallbackPtr
}

// Integer arguments are taken as uintptr and narrowed to culong, because only
// the low 32 bits of a register holding an unsigned long are defined on Windows.
func streamCallbackTrampoline(input, output unsafe.Pointer, frameCount uintptr, timeInfo unsafe.Pointer, statusFlags uintptr, userData uintptr) uintptr {
	s := lookupStream(userData)
	if s == nil || s.callback == nil {
		return uintptr(Abort)
	}

	frames := int(culong(frameCount))
	in := Buffer{ptr: input, frames: frames, channels: s.input.channels, format: s.input.format}
	out := Buffer{ptr: output, frames: frames, channels: s.output.channels, format: s.output.format}
	var ti StreamCallbackTimeInfo
	if timeInfo != nil {
		ti = *(*StreamCallbackTimeInfo)(timeInfo)
	}
	return uintptr(s.callback(in, out, ti, StreamCallbackFlags(culong(statusFlags))))
}

func finishedCallbackTrampoline(userData uintptr) uintptr {
	if s := lookupStream(userData); s != nil {
		if f := s.finished.Load(); f != nil {
			(*f)()
		}
	}
	return 0
}
