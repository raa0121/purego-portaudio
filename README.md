# purego-portaudio

[![Go Reference](https://pkg.go.dev/badge/github.com/raa0121/purego-portaudio.svg)](https://pkg.go.dev/github.com/raa0121/purego-portaudio)

Go bindings for [PortAudio](https://www.portaudio.com/) that need no cgo. The
PortAudio shared library is loaded at run time with
[purego](https://github.com/ebitengine/purego), so `CGO_ENABLED=0` builds and
cross-compilation work.

- The core PortAudio API: host APIs, devices, blocking streams and callback streams
- WASAPI extensions on Windows: loopback capture, stream flags, device mix and
  default formats, and stream state notifications
- Supports 64-bit (amd64 and arm64) Windows, Linux and macOS, with Go 1.25 or later

```sh
go get github.com/raa0121/purego-portaudio
```

## Getting PortAudio

You need the PortAudio shared library at run time. This module does not bundle it.

| Platform | Library | How to get it |
|---|---|---|
| Linux | `libportaudio.so.2` | `apt install libportaudio2`, `dnf install portaudio` |
| macOS | `libportaudio.2.dylib` | `brew install portaudio` |
| Windows | `portaudio.dll` | Build from source, see below |

[`Initialize`](https://pkg.go.dev/github.com/raa0121/purego-portaudio#Initialize)
tries these names. Call
[`Load`](https://pkg.go.dev/github.com/raa0121/purego-portaudio#Load) first to use
another file:

```go
if err := portaudio.Load(`C:\path\to\portaudio.dll`); err != nil {
	log.Fatal(err)
}
```

On Windows, names without a directory are only looked up in the application's
directory and the system directories. The current directory and `PATH` are not
searched. Ship `portaudio.dll` next to your executable, or pass an absolute path.

### Building PortAudio for Windows

WASAPI loopback devices (`"<output name> [Loopback]"`) only exist in PortAudio
builds newer than the v19.7.0 release, so build from the `master` branch.

With CMake and Visual Studio:

```sh
git clone https://github.com/PortAudio/portaudio
cmake -S portaudio -B build -DPA_BUILD_SHARED_LIBS=ON -DPA_USE_ASIO=OFF
cmake --build build --config Release
# build/Release/portaudio.dll
```

With MSYS2 MinGW-w64 and no CMake, the following builds a DLL with the MME and
WASAPI host APIs:

```sh
cd portaudio
gcc -O2 -shared -o portaudio.dll \
  -Iinclude -Isrc/common -Isrc/os/win \
  -DPA_USE_WMME=1 -DPA_USE_WASAPI=1 -DPA_USE_DS=0 -DPA_USE_ASIO=0 \
  -DPA_USE_WDMKS=0 -DPA_USE_JACK=0 -DPA_USE_SKELETON=0 \
  src/common/*.c \
  src/os/win/pa_win_hostapis.c src/os/win/pa_win_util.c \
  src/os/win/pa_win_waveformat.c src/os/win/pa_win_coinitialize.c \
  src/os/win/pa_win_version.c \
  src/hostapi/wmme/pa_win_wmme.c src/hostapi/wasapi/pa_win_wasapi.c \
  -static-libgcc -lwinmm -lole32 -luuid
```

Add `-DPA_WASAPI_MAX_CONST_DEVICE_COUNT=32` to enable
`UpdateWASAPIDeviceList`. That build pads the WASAPI device list with
inactive placeholder devices.

## Usage

### Blocking read

```go
if err := portaudio.Initialize(); err != nil {
	log.Fatal(err)
}
defer portaudio.Terminate()

stream, err := portaudio.OpenDefaultStream(1, 0, portaudio.Float32, 48000, portaudio.FramesPerBufferUnspecified, nil)
if err != nil {
	log.Fatal(err)
}
defer stream.Close()
if err := stream.Start(); err != nil {
	log.Fatal(err)
}

buf := make([]float32, 4800)
for {
	if err := portaudio.ReadStream(stream, buf); err != nil && !errors.Is(err, portaudio.ErrInputOverflowed) {
		log.Fatal(err)
	}
	// process buf
}
```

`ReadStream` and `WriteStream` take `[]float32`, `[]int32`, `[]int16`, `[]int8`
or `[]uint8` matching the stream's sample format, with interleaved channels.
`ReadRaw` and `WriteRaw` take any layout, including `Int24` and `NonInterleaved`.

### Callback

```go
stream, err := portaudio.OpenDefaultStream(0, 2, portaudio.Float32, 48000, 0,
	func(in, out portaudio.Buffer, _ portaudio.StreamCallbackTimeInfo, _ portaudio.StreamCallbackFlags) portaudio.StreamCallbackResult {
		samples := portaudio.Samples[float32](out) // interleaved, len = frames * channels
		for i := range samples {
			samples[i] = 0
		}
		return portaudio.Continue
	})
```

The callback runs on PortAudio's audio thread. It should not block, and the
buffers are only valid until it returns. Use `ChannelSamples` for
`NonInterleaved` buffers.

### WASAPI loopback (Windows)

```go
wasapi, _ := portaudio.HostAPIIndexOf(portaudio.WASAPI)
api, _ := portaudio.HostAPI(wasapi)
loopback, err := portaudio.WASAPILoopbackOf(api.DefaultOutputDevice)
if err != nil {
	log.Fatal(err) // needs a PortAudio build newer than v19.7.0
}
d, _ := portaudio.Device(loopback)

stream, err := portaudio.OpenStream(&portaudio.StreamParameters{
	Device:           loopback,
	ChannelCount:     d.MaxInputChannels,
	SampleFormat:     portaudio.Float32,
	SuggestedLatency: d.DefaultLowInputLatency,
}, nil, d.DefaultSampleRate, 0, portaudio.NoFlag, nil)
```

Loopback streams must use the device's mix format sample rate, which is
`d.DefaultSampleRate`, or set `WASAPIStreamInfo{Flags: WASAPIFlagAutoConvert}`.
When no application is playing audio, a loopback stream may receive no data, and
`ReadStream` blocks until something plays.

See [`examples/`](examples) for complete programs:
[`devices`](examples/devices) lists devices, [`sine`](examples/sine) plays a tone
from a callback, and [`loopback`](examples/loopback) shows the level of what the
speakers are playing.

## Notes

- **Errors.** PortAudio error codes are `portaudio.ErrorCode` values. Compare
  them with `errors.Is(err, portaudio.ErrInvalidDevice)`. Host API failures are
  returned as `*portaudio.HostError` when PortAudio records details.
- **COM on Windows.** Go may run consecutive calls of one goroutine on different
  OS threads, and the WASAPI and DirectSound backends need COM on each of them.
  `Initialize` keeps the process-wide multithreaded apartment alive, so you do not
  need `runtime.LockOSThread`. Host APIs that need a single-threaded apartment,
  such as ASIO, are not supported.
- **State handlers.** `Stream.SetWASAPIStateHandler` is only called for callback
  streams, because blocking streams have no processing thread.
- **Not wrapped.** WASAPI host processor callbacks, `PaWasapi_GetAudioClient`,
  `PaWasapi_GetIMMDevice`, jack descriptions, thread priority boosting, the WinRT
  functions, and the extensions of other host APIs (ASIO, ALSA, Core Audio, JACK).
  You can still pass a host API specific struct with
  `RawHostAPISpecificStreamInfo`.

## Testing

```sh
go test ./...
```

Tests that need PortAudio are skipped when the library cannot be loaded. Set
`PORTAUDIO_LIBRARY` to test a specific build:

```sh
PORTAUDIO_LIBRARY=/path/to/portaudio.dll go test ./...
```

On Windows, the integration tests play a quiet tone at -60 dBFS on the default
output device to check loopback capture.

## License

MIT. See [LICENSE](LICENSE). PortAudio is distributed under its own
[license](https://github.com/PortAudio/portaudio/blob/master/LICENSE.txt).
