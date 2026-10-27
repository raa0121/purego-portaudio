package portaudio

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// StreamParameters describes one direction (input or output) of a stream.
type StreamParameters struct {
	Device       DeviceIndex
	ChannelCount int
	SampleFormat SampleFormat
	// SuggestedLatency is usually DeviceInfo.DefaultLowInputLatency (or the output
	// or high latency equivalents).
	SuggestedLatency time.Duration
	// HostAPISpecificStreamInfo is optional. See WASAPIStreamInfo on Windows.
	HostAPISpecificStreamInfo HostAPISpecificStreamInfo
}

// HostAPISpecificStreamInfo is a host API specific stream description such as
// *WASAPIStreamInfo, or a RawHostAPISpecificStreamInfo for host APIs that this
// package does not wrap.
type HostAPISpecificStreamInfo interface {
	streamInfoPointer() unsafe.Pointer
}

// RawHostAPISpecificStreamInfo passes a pointer to a host API specific C struct
// as is. If it points to Go memory, that memory must not contain Go pointers.
type RawHostAPISpecificStreamInfo struct {
	Pointer unsafe.Pointer
}

func (r RawHostAPISpecificStreamInfo) streamInfoPointer() unsafe.Pointer {
	return r.Pointer
}

type cStreamParameters struct {
	device                    int32
	channelCount              int32
	sampleFormat              culong
	suggestedLatency          float64
	hostAPISpecificStreamInfo unsafe.Pointer
}

func (p *StreamParameters) toC(pinner *runtime.Pinner) *cStreamParameters {
	if p == nil {
		return nil
	}
	c := &cStreamParameters{
		device:           int32(p.Device),
		channelCount:     int32(p.ChannelCount),
		sampleFormat:     culong(p.SampleFormat),
		suggestedLatency: p.SuggestedLatency.Seconds(),
	}
	if p.HostAPISpecificStreamInfo != nil {
		if info := p.HostAPISpecificStreamInfo.streamInfoPointer(); info != nil {
			pinner.Pin(info)
			c.hostAPISpecificStreamInfo = info
		}
	}
	return c
}

func (p *StreamParameters) layout() bufferLayout {
	if p == nil {
		return bufferLayout{}
	}
	return bufferLayout{channels: p.ChannelCount, format: p.SampleFormat}
}

// IsFormatSupported reports, as a nil error, whether a stream with the given
// parameters could be opened. Either in or out may be nil.
func IsFormatSupported(in, out *StreamParameters, sampleRate float64) error {
	if !loaded.Load() {
		return ErrNotInitialized
	}
	var pinner runtime.Pinner
	defer pinner.Unpin()
	return newError(paIsFormatSupported(in.toC(&pinner), out.toC(&pinner), sampleRate))
}

type bufferLayout struct {
	channels int
	format   SampleFormat
}

// Stream is an open PortAudio stream. Its methods must not be called
// concurrently with Close.
type Stream struct {
	ptr            unsafe.Pointer
	id             uintptr
	input, output  bufferLayout
	callback       StreamCallback
	finished       atomic.Pointer[func()]
	wasapiCallback atomic.Value

	// PortAudio's WASAPI backend copies PaStreamParameters, including the
	// hostApiSpecificStreamInfo pointer, into the stream and may read it again
	// later, so that memory stays pinned until Close.
	pinner runtime.Pinner
}

// userData handed to PortAudio is a stream ID instead of a Go pointer, which C
// code is not allowed to retain.
var (
	streams      sync.Map // uintptr -> *Stream
	lastStreamID atomic.Uintptr
)

func lookupStream(id uintptr) *Stream {
	v, ok := streams.Load(id)
	if !ok {
		return nil
	}
	return v.(*Stream)
}

// OpenStream opens a stream. Pass nil for in or out to open an output-only or
// input-only stream.
//
// If callback is nil the stream is opened in blocking mode; use ReadStream and
// WriteStream (or ReadRaw and WriteRaw) to transfer audio. Otherwise callback is
// called on PortAudio's audio thread whenever audio needs to be processed.
//
// The stream is stopped when opened; call Start to begin processing.
func OpenStream(in, out *StreamParameters, sampleRate float64, framesPerBuffer int, flags StreamFlags, callback StreamCallback) (*Stream, error) {
	if !loaded.Load() {
		return nil, ErrNotInitialized
	}
	s := &Stream{
		id:       lastStreamID.Add(1),
		input:    in.layout(),
		output:   out.layout(),
		callback: callback,
	}
	cin := in.toC(&s.pinner)
	cout := out.toC(&s.pinner)

	return s.open(func(ptr *unsafe.Pointer, cb uintptr) int32 {
		return paOpenStream(ptr, cin, cout, sampleRate, culong(framesPerBuffer), culong(flags), cb, s.id)
	})
}

// OpenDefaultStream opens a stream on the default input and/or output device
// with the given number of channels. See OpenStream.
func OpenDefaultStream(numInputChannels, numOutputChannels int, format SampleFormat, sampleRate float64, framesPerBuffer int, callback StreamCallback) (*Stream, error) {
	if !loaded.Load() {
		return nil, ErrNotInitialized
	}
	s := &Stream{
		id:       lastStreamID.Add(1),
		input:    bufferLayout{channels: numInputChannels, format: format},
		output:   bufferLayout{channels: numOutputChannels, format: format},
		callback: callback,
	}

	return s.open(func(ptr *unsafe.Pointer, cb uintptr) int32 {
		return paOpenDefaultStream(ptr, int32(numInputChannels), int32(numOutputChannels), culong(format), sampleRate, culong(framesPerBuffer), cb, s.id)
	})
}

func (s *Stream) open(call func(ptr *unsafe.Pointer, cb uintptr) int32) (*Stream, error) {
	var cb uintptr
	if s.callback != nil {
		cb = streamCallbackAddr()
	}

	streams.Store(s.id, s)
	// A separate variable keeps the Stream itself, which holds Go pointers, from
	// being passed to C.
	var ptr unsafe.Pointer
	if err := newError(call(&ptr, cb)); err != nil {
		streams.Delete(s.id)
		s.pinner.Unpin()
		return nil, err
	}
	s.ptr = ptr
	return s, nil
}

// Close closes the stream, aborting it first if it is active.
func (s *Stream) Close() error {
	if s.ptr == nil {
		return ErrBadStreamPtr
	}
	err := newError(paCloseStream(s.ptr))
	// Terminate has already closed the stream on the C side.
	if err != nil && err != ErrNotInitialized {
		return err
	}
	s.ptr = nil
	streams.Delete(s.id)
	s.pinner.Unpin()
	return err
}

// Start starts audio processing.
func (s *Stream) Start() error {
	return s.call(paStartStream)
}

// Stop stops audio processing after pending output has been played.
func (s *Stream) Stop() error {
	return s.call(paStopStream)
}

// Abort stops audio processing immediately, discarding pending output.
func (s *Stream) Abort() error {
	return s.call(paAbortStream)
}

func (s *Stream) call(f func(unsafe.Pointer) int32) error {
	if s.ptr == nil {
		return ErrBadStreamPtr
	}
	return newError(f(s.ptr))
}

// IsStopped reports whether the stream is stopped, that is, not started or
// stopped with Stop or Abort.
func (s *Stream) IsStopped() (bool, error) {
	return s.boolCall(paIsStreamStopped)
}

// IsActive reports whether the stream is processing audio. A callback stream
// becomes inactive after its callback returns Complete or Abort, even before Stop
// is called.
func (s *Stream) IsActive() (bool, error) {
	return s.boolCall(paIsStreamActive)
}

func (s *Stream) boolCall(f func(unsafe.Pointer) int32) (bool, error) {
	if s.ptr == nil {
		return false, ErrBadStreamPtr
	}
	r := f(s.ptr)
	if r < 0 {
		return false, newError(r)
	}
	return r == 1, nil
}

// StreamInfo describes the actual parameters of an open stream.
type StreamInfo struct {
	InputLatency  time.Duration
	OutputLatency time.Duration
	SampleRate    float64
}

type cStreamInfo struct {
	structVersion int32
	inputLatency  float64
	outputLatency float64
	sampleRate    float64
}

// Info returns the latencies and the sample rate the stream actually uses.
func (s *Stream) Info() (*StreamInfo, error) {
	if s.ptr == nil {
		return nil, ErrBadStreamPtr
	}
	p := paGetStreamInfo(s.ptr)
	if p == nil {
		return nil, ErrBadStreamPtr
	}
	c := (*cStreamInfo)(p)
	return &StreamInfo{
		InputLatency:  secondsToDuration(c.inputLatency),
		OutputLatency: secondsToDuration(c.outputLatency),
		SampleRate:    c.sampleRate,
	}, nil
}

// Time returns the stream's clock in seconds, the same time base as
// StreamCallbackTimeInfo. It returns 0 on error.
func (s *Stream) Time() float64 {
	if s.ptr == nil {
		return 0
	}
	return paGetStreamTime(s.ptr)
}

// CPULoad returns the fraction, from 0 to 1, of the available CPU time spent in
// the stream callback. It is always 0 for blocking streams.
func (s *Stream) CPULoad() float64 {
	if s.ptr == nil {
		return 0
	}
	return paGetStreamCpuLoad(s.ptr)
}

// ReadAvailable returns the number of frames that can be read without blocking.
func (s *Stream) ReadAvailable() (int, error) {
	return s.availableCall(paGetStreamReadAvailable)
}

// WriteAvailable returns the number of frames that can be written without
// blocking.
func (s *Stream) WriteAvailable() (int, error) {
	return s.availableCall(paGetStreamWriteAvailable)
}

func (s *Stream) availableCall(f func(unsafe.Pointer) clong) (int, error) {
	if s.ptr == nil {
		return 0, ErrBadStreamPtr
	}
	n := f(s.ptr)
	if n < 0 {
		return 0, newError(int32(n))
	}
	return int(n), nil
}

// ReadRaw reads frames frames from a blocking stream into buffer, which must be
// laid out as described by the stream's input SampleFormat. For NonInterleaved
// formats, buffer points to an array of one pointer per channel.
func (s *Stream) ReadRaw(buffer unsafe.Pointer, frames int) error {
	if s.ptr == nil {
		return ErrBadStreamPtr
	}
	return newError(paReadStream(s.ptr, buffer, culong(frames)))
}

// WriteRaw writes frames frames from buffer to a blocking stream. See ReadRaw.
func (s *Stream) WriteRaw(buffer unsafe.Pointer, frames int) error {
	if s.ptr == nil {
		return ErrBadStreamPtr
	}
	return newError(paWriteStream(s.ptr, buffer, culong(frames)))
}

// ReadStream reads interleaved samples from a blocking stream, blocking until buf
// is full. len(buf) must be a multiple of the input channel count, and T must
// match the input SampleFormat.
//
// ErrInputOverflowed is returned when input was discarded before this read; buf is
// filled regardless.
func ReadStream[T Sample](s *Stream, buf []T) error {
	frames, err := framesOf(s.input, buf, ErrCanNotReadFromAnOutputOnlyStream)
	if err != nil || frames == 0 {
		return err
	}
	return s.ReadRaw(unsafe.Pointer(unsafe.SliceData(buf)), frames)
}

// WriteStream writes interleaved samples to a blocking stream, blocking until
// all of buf has been queued. len(buf) must be a multiple of the output channel
// count, and T must match the output SampleFormat.
//
// ErrOutputUnderflowed is returned when output ran dry before this write; buf is
// written regardless.
func WriteStream[T Sample](s *Stream, buf []T) error {
	frames, err := framesOf(s.output, buf, ErrCanNotWriteToAnInputOnlyStream)
	if err != nil || frames == 0 {
		return err
	}
	return s.WriteRaw(unsafe.Pointer(unsafe.SliceData(buf)), frames)
}

func framesOf[T Sample](layout bufferLayout, buf []T, noChannels ErrorCode) (int, error) {
	if layout.channels == 0 {
		return 0, noChannels
	}
	if want := sampleFormatOf[T](); layout.format != want {
		return 0, fmt.Errorf("portaudio: buffer of %v does not match the stream sample format %v (use ReadRaw or WriteRaw for other layouts)", want, layout.format)
	}
	if len(buf)%layout.channels != 0 {
		return 0, fmt.Errorf("portaudio: buffer length %d is not a multiple of the channel count %d", len(buf), layout.channels)
	}
	return len(buf) / layout.channels, nil
}

// SetFinishedCallback sets a function that is called on PortAudio's audio thread
// when the stream becomes inactive, or removes it if f is nil. The stream must be
// stopped.
func (s *Stream) SetFinishedCallback(f func()) error {
	if s.ptr == nil {
		return ErrBadStreamPtr
	}
	if f == nil {
		s.finished.Store(nil)
		return newError(paSetStreamFinishedCallback(s.ptr, 0))
	}
	s.finished.Store(&f)
	return newError(paSetStreamFinishedCallback(s.ptr, finishedCallbackAddr()))
}
