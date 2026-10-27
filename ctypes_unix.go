//go:build !windows

package portaudio

// C long and unsigned long are 64 bits on 64-bit Unix (LP64).
type (
	clong  = int64
	culong = uint64
)
