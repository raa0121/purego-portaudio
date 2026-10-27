package portaudio

import (
	"errors"
	"testing"
)

// This file sorts before integration_test.go, so these tests run before the
// library is loaded.
func TestNotLoaded(t *testing.T) {
	if loaded.Load() {
		t.Skip("library already loaded")
	}
	if _, err := DeviceCount(); !errors.Is(err, ErrNotInitialized) {
		t.Errorf("DeviceCount before Load = %v, want ErrNotInitialized", err)
	}
	if got := DefaultOutputDevice(); got != NoDevice {
		t.Errorf("DefaultOutputDevice before Load = %d, want NoDevice", got)
	}
	if got, want := ErrInvalidDevice.Error(), "portaudio: error -9996"; got != want {
		t.Errorf("ErrInvalidDevice.Error() before Load = %q, want %q", got, want)
	}
}

func TestHostErrorUnwrap(t *testing.T) {
	var err error = &HostError{HostAPIType: WASAPI, Code: -1, Text: "x"}
	if !errors.Is(err, ErrUnanticipatedHostError) {
		t.Error("HostError does not match ErrUnanticipatedHostError")
	}
}
