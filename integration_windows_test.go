package portaudio

import (
	"errors"
	"math"
	"sync/atomic"
	"testing"
	"time"
)

func TestWASAPILoopback(t *testing.T) {
	setup(t)
	if paWasapiIsLoopback == nil {
		t.Skip("PortAudio was built without WASAPI loopback support")
	}
	wasapi, err := HostAPIIndexOf(WASAPI)
	if errors.Is(err, ErrHostAPINotFound) {
		t.Skip("WASAPI is not available")
	} else if err != nil {
		t.Fatal(err)
	}
	api, err := HostAPI(wasapi)
	if err != nil {
		t.Fatal(err)
	}
	if api.DefaultOutputDevice == NoDevice {
		t.Skip("no WASAPI output device")
	}

	if ok, err := IsWASAPILoopback(api.DefaultOutputDevice); err != nil || ok {
		t.Errorf("IsWASAPILoopback(default output) = %v, %v, want false", ok, err)
	}
	loopback, err := WASAPILoopbackOf(api.DefaultOutputDevice)
	if err != nil {
		t.Fatal(err)
	}
	d, err := Device(loopback)
	if err != nil {
		t.Fatal(err)
	}
	mix, err := WASAPIDeviceMixFormat(loopback)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("loopback %d: %q, mix format %+v", loopback, d.Name, mix)
	if float64(mix.SamplesPerSec) != d.DefaultSampleRate {
		t.Errorf("mix format rate %d != DefaultSampleRate %v", mix.SamplesPerSec, d.DefaultSampleRate)
	}

	// Loopback capture may deliver nothing while no audio is rendered, so play a
	// -60 dBFS tone on the same endpoint.
	var phase float64
	render, err := OpenStream(nil, &StreamParameters{
		Device:           api.DefaultOutputDevice,
		ChannelCount:     2,
		SampleFormat:     Float32,
		SuggestedLatency: 50 * time.Millisecond,
	}, d.DefaultSampleRate, 0, NoFlag, func(_, out Buffer, _ StreamCallbackTimeInfo, _ StreamCallbackFlags) StreamCallbackResult {
		samples := Samples[float32](out)
		for i := 0; i < len(samples); i += 2 {
			v := float32(1e-3 * math.Sin(phase))
			samples[i], samples[i+1] = v, v
			phase += 2 * math.Pi * 1000 / d.DefaultSampleRate
		}
		return Continue
	})
	if err != nil {
		t.Fatal(err)
	}
	defer render.Close()

	capture, err := OpenStream(&StreamParameters{
		Device:                    loopback,
		ChannelCount:              d.MaxInputChannels,
		SampleFormat:              Float32,
		SuggestedLatency:          d.DefaultLowInputLatency,
		HostAPISpecificStreamInfo: &WASAPIStreamInfo{Flags: WASAPIFlagAutoConvert},
	}, nil, d.DefaultSampleRate, 0, NoFlag, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Close()

	var states atomic.Uint32
	stopped := make(chan struct{})
	if err := render.SetWASAPIStateHandler(func(state WASAPIStreamState, _ uint32) {
		states.Or(uint32(state))
		if state&WASAPIStreamStateThreadStop != 0 {
			close(stopped)
		}
	}); err != nil {
		t.Fatal(err)
	}

	if err := render.Start(); err != nil {
		t.Fatal(err)
	}
	if err := capture.Start(); err != nil {
		t.Fatal(err)
	}

	// ReadStream would block forever if loopback delivered nothing.
	buf := make([]float32, int(d.DefaultSampleRate/10)*d.MaxInputChannels)
	read := make(chan error, 1)
	go func() {
		// The first buffer may predate the tone reaching the mixer.
		if err := ReadStream(capture, buf); err != nil && !errors.Is(err, ErrInputOverflowed) {
			read <- err
			return
		}
		read <- ReadStream(capture, buf)
	}()
	select {
	case err := <-read:
		if err != nil && !errors.Is(err, ErrInputOverflowed) {
			t.Fatalf("ReadStream: %v", err)
		}
	case <-time.After(3 * time.Second):
		capture.Abort()
		t.Fatal("ReadStream on the loopback device did not return")
	}
	var peak float32
	for _, s := range buf {
		peak = max(peak, float32(math.Abs(float64(s))))
	}
	t.Logf("captured peak %.2g", peak)
	if peak == 0 {
		t.Error("loopback captured only silence while a tone was playing")
	}

	if in, _, err := capture.WASAPIFramesPerHostBuffer(); err != nil || in <= 0 {
		t.Errorf("WASAPIFramesPerHostBuffer = %d, %v", in, err)
	}
	if err := capture.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := render.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("state handler did not report ThreadStop")
	}
	want := WASAPIStreamStateThreadPrepare | WASAPIStreamStateThreadStart | WASAPIStreamStateThreadStop
	if got := WASAPIStreamState(states.Load()); got != want {
		t.Errorf("state handler saw %04b, want %04b", got, want)
	}
}
