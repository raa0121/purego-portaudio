// Package portaudio provides Go bindings for PortAudio (https://www.portaudio.com/)
// without cgo, using github.com/ebitengine/purego.
//
// The PortAudio shared library is loaded at run time. [Initialize] loads it from the
// default locations for the platform; call [Load] first to use a specific file.
//
//	if err := portaudio.Initialize(); err != nil {
//		log.Fatal(err)
//	}
//	defer portaudio.Terminate()
//
// Streams can be used in blocking mode with [ReadStream] and [WriteStream], or in
// callback mode by passing a [StreamCallback] to [OpenStream]. Stream callbacks are
// invoked on PortAudio's audio thread.
//
// On Windows the WASAPI extensions (loopback detection, WASAPI stream flags, device
// formats, stream state notifications) are available. WASAPI loopback devices are
// only enumerated by PortAudio builds newer than v19.7.0.
//
// Only 64-bit (amd64 and arm64) Windows, Linux and macOS are supported, because
// the bindings mirror the C struct layouts of those platforms.
package portaudio
