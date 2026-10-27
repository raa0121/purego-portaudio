package portaudio

import (
	"errors"
	"testing"
	"unsafe"
)

func TestSampleFormatString(t *testing.T) {
	tests := []struct {
		format SampleFormat
		want   string
	}{
		{Float32, "Float32"},
		{Int16 | NonInterleaved, "Int16|NonInterleaved"},
		{0, "0x0"},
		{Int8 | 0x100, "Int8|0x100"},
	}
	for _, tt := range tests {
		if got := tt.format.String(); got != tt.want {
			t.Errorf("SampleFormat(0x%x).String() = %q, want %q", uint32(tt.format), got, tt.want)
		}
	}
}

func TestFramesOf(t *testing.T) {
	stereo := bufferLayout{channels: 2, format: Float32}

	if n, err := framesOf(stereo, make([]float32, 8), ErrCanNotReadFromAnOutputOnlyStream); err != nil || n != 4 {
		t.Errorf("framesOf(stereo, 8 samples) = %d, %v, want 4, nil", n, err)
	}
	if _, err := framesOf(stereo, make([]float32, 7), ErrCanNotReadFromAnOutputOnlyStream); err == nil {
		t.Error("framesOf with a partial frame succeeded")
	}
	if _, err := framesOf(stereo, make([]int16, 8), ErrCanNotReadFromAnOutputOnlyStream); err == nil {
		t.Error("framesOf with a mismatched sample type succeeded")
	}
	if _, err := framesOf(bufferLayout{}, make([]float32, 8), ErrCanNotReadFromAnOutputOnlyStream); !errors.Is(err, ErrCanNotReadFromAnOutputOnlyStream) {
		t.Errorf("framesOf without channels = %v, want ErrCanNotReadFromAnOutputOnlyStream", err)
	}
}

func TestBufferSamples(t *testing.T) {
	data := []int16{1, 2, 3, 4, 5, 6}
	b := Buffer{ptr: unsafe.Pointer(&data[0]), frames: 3, channels: 2, format: Int16}

	got := Samples[int16](b)
	if len(got) != 6 || got[5] != 6 {
		t.Errorf("Samples = %v, want %v", got, data)
	}
	if Samples[int16](Buffer{}) != nil {
		t.Error("Samples of an empty buffer is not nil")
	}
	assertPanics(t, "Samples with a mismatched type", func() { Samples[float32](b) })
}

func TestBufferChannelSamples(t *testing.T) {
	left := []float32{1, 2, 3}
	right := []float32{4, 5, 6}
	channels := []unsafe.Pointer{unsafe.Pointer(&left[0]), unsafe.Pointer(&right[0])}
	b := Buffer{ptr: unsafe.Pointer(&channels[0]), frames: 3, channels: 2, format: Float32 | NonInterleaved}

	if got := ChannelSamples[float32](b, 1); len(got) != 3 || got[0] != 4 {
		t.Errorf("ChannelSamples(1) = %v, want %v", got, right)
	}
	assertPanics(t, "ChannelSamples out of range", func() { ChannelSamples[float32](b, 2) })
	assertPanics(t, "Samples of a non-interleaved buffer", func() { Samples[float32](b) })
}

func assertPanics(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", name)
		}
	}()
	f()
}
