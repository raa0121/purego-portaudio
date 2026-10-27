package portaudio

import (
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// setup initializes PortAudio for a test, skipping it when the library is not
// available. Set PORTAUDIO_LIBRARY to test a specific build.
func setup(t *testing.T) {
	t.Helper()
	var candidates []string
	if path := os.Getenv("PORTAUDIO_LIBRARY"); path != "" {
		candidates = []string{path}
	}
	if err := Load(candidates...); err != nil {
		t.Skipf("PortAudio is not available: %v", err)
	}
	if err := Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Cleanup(func() {
		if err := Terminate(); err != nil {
			t.Errorf("Terminate: %v", err)
		}
	})
}

func defaultOutput(t *testing.T) *DeviceInfo {
	t.Helper()
	index := DefaultOutputDevice()
	if index == NoDevice {
		t.Skip("no default output device")
	}
	d, err := Device(index)
	if err != nil {
		t.Fatalf("Device(%d): %v", index, err)
	}
	return d
}

func TestVersion(t *testing.T) {
	setup(t)
	v, err := Version()
	if err != nil {
		t.Fatal(err)
	}
	if v.Major != 19 || v.Text == "" {
		t.Errorf("Version() = %+v, want PortAudio 19.x", v)
	}
	t.Logf("%s (revision %s)", v.Text, v.ControlRevision)
}

func TestEnumerate(t *testing.T) {
	setup(t)
	apis, err := HostAPIs()
	if err != nil {
		t.Fatal(err)
	}
	devices, err := Devices()
	if err != nil {
		t.Fatal(err)
	}

	total := 0
	for _, api := range apis {
		t.Logf("host API %d: %s (%v), %d devices", api.Index, api.Name, api.Type, api.DeviceCount)
		total += api.DeviceCount
		for i := range api.DeviceCount {
			index, err := DeviceIndexOf(api.Index, i)
			if err != nil {
				t.Fatal(err)
			}
			if devices[index].HostAPI != api.Index {
				t.Errorf("device %d belongs to host API %d, want %d", index, devices[index].HostAPI, api.Index)
			}
		}
		if got, err := HostAPIIndexOf(api.Type); err != nil || got != api.Index {
			t.Errorf("HostAPIIndexOf(%v) = %d, %v, want %d", api.Type, got, err, api.Index)
		}
	}
	if total != len(devices) {
		t.Errorf("host APIs list %d devices, DeviceCount is %d", total, len(devices))
	}
	for _, d := range devices {
		t.Logf("device %d: %q in=%d out=%d %.0f Hz", d.Index, d.Name, d.MaxInputChannels, d.MaxOutputChannels, d.DefaultSampleRate)
	}
}

func TestErrors(t *testing.T) {
	setup(t)
	_, err := OpenStream(nil, &StreamParameters{Device: 1 << 20, ChannelCount: 1, SampleFormat: Float32}, 48000, 0, NoFlag, nil)
	if !errors.Is(err, ErrInvalidDevice) {
		t.Errorf("OpenStream on a missing device = %v, want ErrInvalidDevice", err)
	}
	if got := ErrInvalidDevice.Error(); got != "portaudio: Invalid device" {
		t.Errorf("ErrInvalidDevice.Error() = %q", got)
	}
	if n, err := SampleSize(Int24); err != nil || n != 3 {
		t.Errorf("SampleSize(Int24) = %d, %v, want 3, nil", n, err)
	}
}

func TestBlockingOutput(t *testing.T) {
	setup(t)
	d := defaultOutput(t)
	out := &StreamParameters{
		Device:           d.Index,
		ChannelCount:     min(d.MaxOutputChannels, 2),
		SampleFormat:     Float32,
		SuggestedLatency: d.DefaultHighOutputLatency,
	}
	if err := IsFormatSupported(nil, out, d.DefaultSampleRate); err != nil {
		t.Fatalf("IsFormatSupported: %v", err)
	}

	s, err := OpenStream(nil, out, d.DefaultSampleRate, FramesPerBufferUnspecified, NoFlag, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	info, err := s.Info()
	if err != nil {
		t.Fatal(err)
	}
	if info.SampleRate != d.DefaultSampleRate {
		t.Errorf("stream sample rate = %v, want %v", info.SampleRate, d.DefaultSampleRate)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	silence := make([]float32, int(d.DefaultSampleRate/10)*out.ChannelCount)
	if err := WriteStream(s, silence); err != nil && !errors.Is(err, ErrOutputUnderflowed) {
		t.Errorf("WriteStream: %v", err)
	}
	if err := ReadStream(s, silence); !errors.Is(err, ErrCanNotReadFromAnOutputOnlyStream) {
		t.Errorf("ReadStream on an output stream = %v, want ErrCanNotReadFromAnOutputOnlyStream", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); !errors.Is(err, ErrBadStreamPtr) {
		t.Errorf("Start after Close = %v, want ErrBadStreamPtr", err)
	}
}

func TestCallbackOutput(t *testing.T) {
	setup(t)
	defaultOutput(t)

	var calls, frames atomic.Int64
	finished := make(chan struct{})
	s, err := OpenDefaultStream(0, 2, Float32, 48000, 256, func(in, out Buffer, _ StreamCallbackTimeInfo, _ StreamCallbackFlags) StreamCallbackResult {
		samples := Samples[float32](out)
		clear(samples)
		if in.Pointer() != nil || len(samples) != out.Frames()*2 {
			return Abort
		}
		calls.Add(1)
		if frames.Add(int64(out.Frames())) >= 48000/5 {
			return Complete
		}
		return Continue
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.SetFinishedCallback(func() { close(finished) }); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not finish")
	}
	if active, err := s.IsActive(); err != nil || active {
		t.Errorf("IsActive after Complete = %v, %v, want false", active, err)
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if calls.Load() == 0 {
		t.Error("callback was never called")
	}
	t.Logf("%d callbacks, %d frames", calls.Load(), frames.Load())
}
