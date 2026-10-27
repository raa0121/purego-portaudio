//go:build !windows

package portaudio

func initialize() (int32, error) {
	return paInitialize(), nil
}

func decrementMTA() {}
