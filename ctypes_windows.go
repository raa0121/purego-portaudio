package portaudio

// C long and unsigned long are 32 bits on Windows (LLP64).
type (
	clong  = int32
	culong = uint32
)
