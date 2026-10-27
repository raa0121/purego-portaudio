package portaudio

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	paGetVersion                      func() int32
	paGetVersionInfo                  func() unsafe.Pointer
	paGetErrorText                    func(code int32) string
	paInitialize                      func() int32
	paTerminate                       func() int32
	paGetHostApiCount                 func() int32
	paGetDefaultHostApi               func() int32
	paGetHostApiInfo                  func(hostAPI int32) unsafe.Pointer
	paHostApiTypeIdToHostApiIndex     func(typeID int32) int32
	paHostApiDeviceIndexToDeviceIndex func(hostAPI, hostAPIDeviceIndex int32) int32
	paGetLastHostErrorInfo            func() unsafe.Pointer
	paGetDeviceCount                  func() int32
	paGetDefaultInputDevice           func() int32
	paGetDefaultOutputDevice          func() int32
	paGetDeviceInfo                   func(device int32) unsafe.Pointer
	paIsFormatSupported               func(in, out *cStreamParameters, sampleRate float64) int32
	paOpenStream                      func(stream *unsafe.Pointer, in, out *cStreamParameters, sampleRate float64, framesPerBuffer, flags culong, callback, userData uintptr) int32
	paOpenDefaultStream               func(stream *unsafe.Pointer, numInputChannels, numOutputChannels int32, format culong, sampleRate float64, framesPerBuffer culong, callback, userData uintptr) int32
	paCloseStream                     func(stream unsafe.Pointer) int32
	paSetStreamFinishedCallback       func(stream unsafe.Pointer, callback uintptr) int32
	paStartStream                     func(stream unsafe.Pointer) int32
	paStopStream                      func(stream unsafe.Pointer) int32
	paAbortStream                     func(stream unsafe.Pointer) int32
	paIsStreamStopped                 func(stream unsafe.Pointer) int32
	paIsStreamActive                  func(stream unsafe.Pointer) int32
	paGetStreamInfo                   func(stream unsafe.Pointer) unsafe.Pointer
	paGetStreamTime                   func(stream unsafe.Pointer) float64
	paGetStreamCpuLoad                func(stream unsafe.Pointer) float64
	paReadStream                      func(stream, buffer unsafe.Pointer, frames culong) int32
	paWriteStream                     func(stream, buffer unsafe.Pointer, frames culong) int32
	paGetStreamReadAvailable          func(stream unsafe.Pointer) clong
	paGetStreamWriteAvailable         func(stream unsafe.Pointer) clong
	paGetSampleSize                   func(format culong) int32
	paSleep                           func(msec clong)
)

type symbol struct {
	name string
	fptr any
	// Optional symbols are left nil when missing, so that older or
	// differently configured builds of PortAudio can still be loaded.
	optional bool
}

var symbols = []symbol{
	{name: "Pa_GetVersion", fptr: &paGetVersion},
	{name: "Pa_GetVersionInfo", fptr: &paGetVersionInfo},
	{name: "Pa_GetErrorText", fptr: &paGetErrorText},
	{name: "Pa_Initialize", fptr: &paInitialize},
	{name: "Pa_Terminate", fptr: &paTerminate},
	{name: "Pa_GetHostApiCount", fptr: &paGetHostApiCount},
	{name: "Pa_GetDefaultHostApi", fptr: &paGetDefaultHostApi},
	{name: "Pa_GetHostApiInfo", fptr: &paGetHostApiInfo},
	{name: "Pa_HostApiTypeIdToHostApiIndex", fptr: &paHostApiTypeIdToHostApiIndex},
	{name: "Pa_HostApiDeviceIndexToDeviceIndex", fptr: &paHostApiDeviceIndexToDeviceIndex},
	{name: "Pa_GetLastHostErrorInfo", fptr: &paGetLastHostErrorInfo},
	{name: "Pa_GetDeviceCount", fptr: &paGetDeviceCount},
	{name: "Pa_GetDefaultInputDevice", fptr: &paGetDefaultInputDevice},
	{name: "Pa_GetDefaultOutputDevice", fptr: &paGetDefaultOutputDevice},
	{name: "Pa_GetDeviceInfo", fptr: &paGetDeviceInfo},
	{name: "Pa_IsFormatSupported", fptr: &paIsFormatSupported},
	{name: "Pa_OpenStream", fptr: &paOpenStream},
	{name: "Pa_OpenDefaultStream", fptr: &paOpenDefaultStream},
	{name: "Pa_CloseStream", fptr: &paCloseStream},
	{name: "Pa_SetStreamFinishedCallback", fptr: &paSetStreamFinishedCallback},
	{name: "Pa_StartStream", fptr: &paStartStream},
	{name: "Pa_StopStream", fptr: &paStopStream},
	{name: "Pa_AbortStream", fptr: &paAbortStream},
	{name: "Pa_IsStreamStopped", fptr: &paIsStreamStopped},
	{name: "Pa_IsStreamActive", fptr: &paIsStreamActive},
	{name: "Pa_GetStreamInfo", fptr: &paGetStreamInfo},
	{name: "Pa_GetStreamTime", fptr: &paGetStreamTime},
	{name: "Pa_GetStreamCpuLoad", fptr: &paGetStreamCpuLoad},
	{name: "Pa_ReadStream", fptr: &paReadStream},
	{name: "Pa_WriteStream", fptr: &paWriteStream},
	{name: "Pa_GetStreamReadAvailable", fptr: &paGetStreamReadAvailable},
	{name: "Pa_GetStreamWriteAvailable", fptr: &paGetStreamWriteAvailable},
	{name: "Pa_GetSampleSize", fptr: &paGetSampleSize},
	{name: "Pa_Sleep", fptr: &paSleep},
}

var (
	loadMu sync.Mutex
	loaded atomic.Bool
)

// Load loads the PortAudio shared library and resolves its functions.
//
// The candidates are tried in order and the first one that loads is used. With no
// arguments, the default names for the platform are tried: portaudio.dll and
// libportaudio*.dll on Windows (searched in the application directory and the
// system directories, not in the current directory or PATH), libportaudio.so.2 on
// Linux, and libportaudio.2.dylib (including the Homebrew prefixes) on macOS.
//
// Once a library has been loaded, further calls do nothing and return nil.
// [Initialize] calls Load with no arguments if no library has been loaded yet.
func Load(candidates ...string) error {
	loadMu.Lock()
	defer loadMu.Unlock()

	if loaded.Load() {
		return nil
	}
	if len(candidates) == 0 {
		candidates = defaultLibraryNames()
	}

	var errs []error
	for _, name := range candidates {
		if err := loadLibrary(name); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		loaded.Store(true)
		return nil
	}
	return fmt.Errorf("portaudio: failed to load the PortAudio library: %w", errors.Join(errs...))
}

func loadLibrary(name string) error {
	handle, err := openLibrary(name)
	if err != nil {
		return err
	}

	// Resolve everything before registering, so that a library missing a required
	// symbol leaves no function half-registered.
	addrs := make([]uintptr, len(symbols))
	for i, s := range symbols {
		addr, err := lookupSymbol(handle, s.name)
		if err != nil || addr == 0 {
			if s.optional {
				continue
			}
			closeLibrary(handle)
			return errMissingSymbol(s.name)
		}
		addrs[i] = addr
	}

	for i, s := range symbols {
		if addrs[i] != 0 {
			purego.RegisterFunc(s.fptr, addrs[i])
		}
	}
	return nil
}

func errMissingSymbol(name string) error {
	return fmt.Errorf("portaudio: %s is not exported by the loaded PortAudio library: %w", name, errors.ErrUnsupported)
}
