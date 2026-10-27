//go:build !((windows || linux || darwin) && (amd64 || arm64))

package portaudio

// The Go mirrors of PortAudio's structs assume the LLP64/LP64 layouts of these
// platforms, so building elsewhere would silently corrupt memory.
var _ = purego_portaudio_supports_only_64bit_windows_linux_darwin
