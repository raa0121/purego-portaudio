//go:build !windows

package portaudio

func platformLayoutTests() []struct {
	name string
	got  uintptr
	want uintptr
} {
	return nil
}
