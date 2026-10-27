package portaudio

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
)

var (
	paWasapiIsLoopback             func(device int32) int32
	paWasapiGetDeviceRole          func(device int32) int32
	paWasapiUpdateDeviceList       func() int32
	paWasapiGetDeviceDefaultFormat func(format unsafe.Pointer, formatSize uint32, device int32) int32
	paWasapiGetDeviceMixFormat     func(format unsafe.Pointer, formatSize uint32, device int32) int32
	paWasapiGetDeviceCurrentFormat func(stream, format unsafe.Pointer, formatSize uint32, output int32) int32
	paWasapiGetFramesPerHostBuffer func(stream unsafe.Pointer, input, output *uint32) int32
	paWasapiSetStreamStateHandler  func(stream unsafe.Pointer, handler, userData uintptr) int32
)

func init() {
	symbols = append(symbols,
		symbol{name: "PaWasapi_IsLoopback", fptr: &paWasapiIsLoopback, optional: true},
		symbol{name: "PaWasapi_GetDeviceRole", fptr: &paWasapiGetDeviceRole, optional: true},
		symbol{name: "PaWasapi_UpdateDeviceList", fptr: &paWasapiUpdateDeviceList, optional: true},
		symbol{name: "PaWasapi_GetDeviceDefaultFormat", fptr: &paWasapiGetDeviceDefaultFormat, optional: true},
		symbol{name: "PaWasapi_GetDeviceMixFormat", fptr: &paWasapiGetDeviceMixFormat, optional: true},
		symbol{name: "PaWasapi_GetDeviceCurrentFormat", fptr: &paWasapiGetDeviceCurrentFormat, optional: true},
		symbol{name: "PaWasapi_GetFramesPerHostBuffer", fptr: &paWasapiGetFramesPerHostBuffer, optional: true},
		symbol{name: "PaWasapi_SetStreamStateHandler", fptr: &paWasapiSetStreamStateHandler, optional: true},
	)
}

// WASAPIFlags are WASAPI specific stream flags (PaWasapiFlags).
//
// paWinWasapiRedirectHostProcessor is not provided, because host processor
// callbacks are not supported.
type WASAPIFlags uint32

const (
	// WASAPIFlagExclusive opens the device in exclusive mode.
	WASAPIFlagExclusive WASAPIFlags = 1 << 0
	// WASAPIFlagUseChannelMask applies WASAPIStreamInfo.ChannelMask.
	WASAPIFlagUseChannelMask WASAPIFlags = 1 << 2
	// WASAPIFlagPolling uses polling instead of event driven processing.
	WASAPIFlagPolling WASAPIFlags = 1 << 3
	// WASAPIFlagThreadPriority applies WASAPIStreamInfo.ThreadPriority.
	WASAPIFlagThreadPriority WASAPIFlags = 1 << 4
	// WASAPIFlagExplicitSampleFormat fails instead of letting PortAudio pick a
	// working format when the requested one is not supported.
	WASAPIFlagExplicitSampleFormat WASAPIFlags = 1 << 5
	// WASAPIFlagAutoConvert lets the system convert channel layouts and sample
	// rates that differ from the mixer's in shared mode.
	WASAPIFlagAutoConvert WASAPIFlags = 1 << 6
	// WASAPIFlagPassthrough sends encoded audio described by
	// WASAPIStreamInfo.Passthrough.
	WASAPIFlagPassthrough WASAPIFlags = 1 << 7
)

// WASAPIThreadPriority is an MMCSS thread priority class (PaWasapiThreadPriority).
type WASAPIThreadPriority int32

const (
	WASAPIThreadPriorityNone          WASAPIThreadPriority = 0
	WASAPIThreadPriorityAudio         WASAPIThreadPriority = 1
	WASAPIThreadPriorityCapture       WASAPIThreadPriority = 2
	WASAPIThreadPriorityDistribution  WASAPIThreadPriority = 3
	WASAPIThreadPriorityGames         WASAPIThreadPriority = 4
	WASAPIThreadPriorityPlayback      WASAPIThreadPriority = 5
	WASAPIThreadPriorityProAudio      WASAPIThreadPriority = 6
	WASAPIThreadPriorityWindowManager WASAPIThreadPriority = 7
)

// WASAPIStreamCategory is an AUDIO_STREAM_CATEGORY (PaWasapiStreamCategory).
type WASAPIStreamCategory int32

const (
	WASAPICategoryOther          WASAPIStreamCategory = 0
	WASAPICategoryCommunications WASAPIStreamCategory = 3
	WASAPICategoryAlerts         WASAPIStreamCategory = 4
	WASAPICategorySoundEffects   WASAPIStreamCategory = 5
	WASAPICategoryGameEffects    WASAPIStreamCategory = 6
	WASAPICategoryGameMedia      WASAPIStreamCategory = 7
	WASAPICategoryGameChat       WASAPIStreamCategory = 8
	WASAPICategorySpeech         WASAPIStreamCategory = 9
	WASAPICategoryMovie          WASAPIStreamCategory = 10
	WASAPICategoryMedia          WASAPIStreamCategory = 11
)

// WASAPIStreamOption is an AUDCLNT_STREAMOPTIONS value (PaWasapiStreamOption).
type WASAPIStreamOption int32

const (
	WASAPIStreamOptionNone WASAPIStreamOption = 0
	// WASAPIStreamOptionRaw bypasses the audio engine's signal processing.
	WASAPIStreamOptionRaw WASAPIStreamOption = 1
	// WASAPIStreamOptionMatchFormat forces the audio engine into the stream format.
	WASAPIStreamOptionMatchFormat WASAPIStreamOption = 2
)

// WASAPIPassthroughFormat identifies an IEC 61937 encoding
// (PaWasapiPassthroughFormat).
type WASAPIPassthroughFormat uint32

const (
	WASAPIPassthroughPcmIec60958           WASAPIPassthroughFormat = 0x00000000
	WASAPIPassthroughDolbyDigital          WASAPIPassthroughFormat = 0x00920000
	WASAPIPassthroughMpeg1                 WASAPIPassthroughFormat = 0x00030cea
	WASAPIPassthroughMpeg3                 WASAPIPassthroughFormat = 0x00040cea
	WASAPIPassthroughMpeg2                 WASAPIPassthroughFormat = 0x00050cea
	WASAPIPassthroughAac                   WASAPIPassthroughFormat = 0x00060cea
	WASAPIPassthroughDts                   WASAPIPassthroughFormat = 0x00080cea
	WASAPIPassthroughDolbyDigitalPlus      WASAPIPassthroughFormat = 0x000a0cea
	WASAPIPassthroughDolbyDigitalPlusAtmos WASAPIPassthroughFormat = 0x010a0cea
	WASAPIPassthroughDtsHd                 WASAPIPassthroughFormat = 0x000b0cea
	WASAPIPassthroughDtsXE1                WASAPIPassthroughFormat = 0x010b0cea
	WASAPIPassthroughDtsXE2                WASAPIPassthroughFormat = 0x030b0cea
	WASAPIPassthroughDolbyMlp              WASAPIPassthroughFormat = 0x000c0cea
	WASAPIPassthroughDolbyMat20            WASAPIPassthroughFormat = 0x010c0cea
	WASAPIPassthroughDolbyMat21            WASAPIPassthroughFormat = 0x030c0cea
	WASAPIPassthroughWmaPro                WASAPIPassthroughFormat = 0x01640000
	WASAPIPassthroughAtrac                 WASAPIPassthroughFormat = 0x00080cea
	WASAPIPassthroughOneBitAudio           WASAPIPassthroughFormat = 0x00090cea
	WASAPIPassthroughDst                   WASAPIPassthroughFormat = 0x000d0cea
)

// WASAPIPassthrough describes encoded audio for WASAPIFlagPassthrough
// (PaWasapiStreamPassthrough).
type WASAPIPassthrough struct {
	FormatID             WASAPIPassthroughFormat
	EncodedSamplesPerSec uint32
	EncodedChannelCount  uint32
	AverageBytesPerSec   uint32
}

// WASAPIStreamInfo holds WASAPI specific stream settings. Set it as
// StreamParameters.HostAPISpecificStreamInfo of a WASAPI device.
type WASAPIStreamInfo struct {
	Flags WASAPIFlags
	// ChannelMask is a WAVEFORMATEXTENSIBLE speaker mask, used with
	// WASAPIFlagUseChannelMask.
	ChannelMask uint32
	// ThreadPriority is used with WASAPIFlagThreadPriority.
	ThreadPriority WASAPIThreadPriority
	StreamCategory WASAPIStreamCategory
	StreamOption   WASAPIStreamOption
	// Passthrough is used with WASAPIFlagPassthrough.
	Passthrough WASAPIPassthrough
}

type cWASAPIStreamInfo struct {
	size                uint32
	hostAPIType         int32
	version             uint32
	flags               uint32
	channelMask         uint32
	hostProcessorOutput uintptr
	hostProcessorInput  uintptr
	threadPriority      int32
	streamCategory      int32
	streamOption        int32
	passthrough         WASAPIPassthrough
}

func (i *WASAPIStreamInfo) streamInfoPointer() unsafe.Pointer {
	return unsafe.Pointer(&cWASAPIStreamInfo{
		// PortAudio rejects the struct unless size and version match exactly.
		size:           uint32(unsafe.Sizeof(cWASAPIStreamInfo{})),
		hostAPIType:    int32(WASAPI),
		version:        1,
		flags:          uint32(i.Flags),
		channelMask:    i.ChannelMask,
		threadPriority: int32(i.ThreadPriority),
		streamCategory: int32(i.StreamCategory),
		streamOption:   int32(i.StreamOption),
		passthrough:    i.Passthrough,
	})
}

// WASAPIDeviceRole is the form factor of a WASAPI device (PaWasapiDeviceRole).
type WASAPIDeviceRole int32

const (
	WASAPIRoleRemoteNetworkDevice       WASAPIDeviceRole = 0
	WASAPIRoleSpeakers                  WASAPIDeviceRole = 1
	WASAPIRoleLineLevel                 WASAPIDeviceRole = 2
	WASAPIRoleHeadphones                WASAPIDeviceRole = 3
	WASAPIRoleMicrophone                WASAPIDeviceRole = 4
	WASAPIRoleHeadset                   WASAPIDeviceRole = 5
	WASAPIRoleHandset                   WASAPIDeviceRole = 6
	WASAPIRoleUnknownDigitalPassthrough WASAPIDeviceRole = 7
	WASAPIRoleSPDIF                     WASAPIDeviceRole = 8
	WASAPIRoleHDMI                      WASAPIDeviceRole = 9
	WASAPIRoleUnknownFormFactor         WASAPIDeviceRole = 10
)

// IsWASAPILoopback reports whether device is a WASAPI loopback device, which
// captures what a render endpoint is playing. device must belong to the WASAPI
// host API.
func IsWASAPILoopback(device DeviceIndex) (bool, error) {
	if !loaded.Load() {
		return false, ErrNotInitialized
	}
	if paWasapiIsLoopback == nil {
		return false, errMissingSymbol("PaWasapi_IsLoopback")
	}
	r := paWasapiIsLoopback(int32(device))
	if r < 0 {
		return false, newError(r)
	}
	return r == 1, nil
}

// PortAudio documents this suffix as stable so that applications can rely on it.
const wasapiLoopbackSuffix = " [Loopback]"

// WASAPILoopbackOf returns the loopback device that captures the WASAPI output
// device output.
func WASAPILoopbackOf(output DeviceIndex) (DeviceIndex, error) {
	target, err := Device(output)
	if err != nil {
		return NoDevice, err
	}
	devices, err := Devices()
	if err != nil {
		return NoDevice, err
	}
	for _, d := range devices {
		if d.HostAPI != target.HostAPI || d.Name != target.Name+wasapiLoopbackSuffix {
			continue
		}
		if ok, err := IsWASAPILoopback(d.Index); err != nil {
			return NoDevice, err
		} else if ok {
			return d.Index, nil
		}
	}
	return NoDevice, fmt.Errorf("portaudio: no WASAPI loopback device for %q: %w", strings.TrimSpace(target.Name), ErrDeviceUnavailable)
}

// WASAPIRole returns the form factor of the WASAPI device.
func WASAPIRole(device DeviceIndex) (WASAPIDeviceRole, error) {
	if !loaded.Load() {
		return 0, ErrNotInitialized
	}
	if paWasapiGetDeviceRole == nil {
		return 0, errMissingSymbol("PaWasapi_GetDeviceRole")
	}
	r := paWasapiGetDeviceRole(int32(device))
	if r < 0 {
		return 0, newError(r)
	}
	return WASAPIDeviceRole(r), nil
}

// UpdateWASAPIDeviceList refreshes the WASAPI device list without restarting
// PortAudio. It only works when PortAudio was built with
// PA_WASAPI_MAX_CONST_DEVICE_COUNT > 0; otherwise call Terminate and Initialize.
func UpdateWASAPIDeviceList() error {
	if !loaded.Load() {
		return ErrNotInitialized
	}
	if paWasapiUpdateDeviceList == nil {
		return errMissingSymbol("PaWasapi_UpdateDeviceList")
	}
	return newError(paWasapiUpdateDeviceList())
}

// WaveFormatExtensible is the FormatTag of a WAVEFORMATEXTENSIBLE.
const WaveFormatExtensible = 0xFFFE

// WaveFormat is a decoded WAVEFORMATEX or WAVEFORMATEXTENSIBLE.
type WaveFormat struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16

	// The following fields are only set when FormatTag is WaveFormatExtensible.
	ValidBitsPerSample uint16
	ChannelMask        uint32
	SubFormat          windows.GUID
}

// sizeof(WAVEFORMATEX) without padding, and sizeof(WAVEFORMATEXTENSIBLE).
const (
	waveFormatExSize         = 18
	waveFormatExtensibleSize = 40
)

func readWaveFormat(read func(buf unsafe.Pointer, size uint32) int32) (*WaveFormat, error) {
	var buf [waveFormatExtensibleSize]byte
	n := read(unsafe.Pointer(&buf[0]), uint32(len(buf)))
	if n < 0 {
		return nil, newError(n)
	}
	if n < waveFormatExSize {
		return nil, fmt.Errorf("portaudio: wave format too short (%d bytes)", n)
	}
	le := binary.LittleEndian
	f := &WaveFormat{
		FormatTag:      le.Uint16(buf[0:]),
		Channels:       le.Uint16(buf[2:]),
		SamplesPerSec:  le.Uint32(buf[4:]),
		AvgBytesPerSec: le.Uint32(buf[8:]),
		BlockAlign:     le.Uint16(buf[12:]),
		BitsPerSample:  le.Uint16(buf[14:]),
	}
	if f.FormatTag == WaveFormatExtensible && n >= waveFormatExtensibleSize {
		f.ValidBitsPerSample = le.Uint16(buf[18:])
		f.ChannelMask = le.Uint32(buf[20:])
		f.SubFormat = windows.GUID{
			Data1: le.Uint32(buf[24:]),
			Data2: le.Uint16(buf[28:]),
			Data3: le.Uint16(buf[30:]),
		}
		copy(f.SubFormat.Data4[:], buf[32:40])
	}
	return f, nil
}

// WASAPIDeviceDefaultFormat returns the shared mode default format of the device
// (PKEY_AudioEngine_DeviceFormat).
func WASAPIDeviceDefaultFormat(device DeviceIndex) (*WaveFormat, error) {
	if !loaded.Load() {
		return nil, ErrNotInitialized
	}
	if paWasapiGetDeviceDefaultFormat == nil {
		return nil, errMissingSymbol("PaWasapi_GetDeviceDefaultFormat")
	}
	return readWaveFormat(func(buf unsafe.Pointer, size uint32) int32 {
		return paWasapiGetDeviceDefaultFormat(buf, size, int32(device))
	})
}

// WASAPIDeviceMixFormat returns the shared mode mix format of the device
// (IAudioClient::GetMixFormat). Loopback streams run at its sample rate.
func WASAPIDeviceMixFormat(device DeviceIndex) (*WaveFormat, error) {
	if !loaded.Load() {
		return nil, ErrNotInitialized
	}
	if paWasapiGetDeviceMixFormat == nil {
		return nil, errMissingSymbol("PaWasapi_GetDeviceMixFormat")
	}
	return readWaveFormat(func(buf unsafe.Pointer, size uint32) int32 {
		return paWasapiGetDeviceMixFormat(buf, size, int32(device))
	})
}

// WASAPICurrentFormat returns the format the device of the stream's input (or
// output if output is true) is running with.
func (s *Stream) WASAPICurrentFormat(output bool) (*WaveFormat, error) {
	if s.ptr == nil {
		return nil, ErrBadStreamPtr
	}
	if paWasapiGetDeviceCurrentFormat == nil {
		return nil, errMissingSymbol("PaWasapi_GetDeviceCurrentFormat")
	}
	var out int32
	if output {
		out = 1
	}
	return readWaveFormat(func(buf unsafe.Pointer, size uint32) int32 {
		return paWasapiGetDeviceCurrentFormat(s.ptr, buf, size, out)
	})
}

// WASAPIFramesPerHostBuffer returns the size in frames of the WASAPI host buffers
// of the stream.
func (s *Stream) WASAPIFramesPerHostBuffer() (input, output int, err error) {
	if s.ptr == nil {
		return 0, 0, ErrBadStreamPtr
	}
	if paWasapiGetFramesPerHostBuffer == nil {
		return 0, 0, errMissingSymbol("PaWasapi_GetFramesPerHostBuffer")
	}
	var in, out uint32
	if err := newError(paWasapiGetFramesPerHostBuffer(s.ptr, &in, &out)); err != nil {
		return 0, 0, err
	}
	return int(in), int(out), nil
}

// WASAPIStreamState is a set of WASAPI stream state flags (PaWasapiStreamState).
type WASAPIStreamState uint32

const (
	// WASAPIStreamStateError is set when the state change was caused by an error,
	// for example AUDCLNT_E_RESOURCES_INVALIDATED after the device was removed.
	WASAPIStreamStateError         WASAPIStreamState = 1 << 0
	WASAPIStreamStateThreadPrepare WASAPIStreamState = 1 << 1
	WASAPIStreamStateThreadStart   WASAPIStreamState = 1 << 2
	WASAPIStreamStateThreadStop    WASAPIStreamState = 1 << 3
)

// WASAPIStateHandler receives WASAPI stream state changes. hresult is the system
// error code when state includes WASAPIStreamStateError.
type WASAPIStateHandler func(state WASAPIStreamState, hresult uint32)

var (
	wasapiStateCallbackOnce sync.Once
	wasapiStateCallbackPtr  uintptr
)

// SetWASAPIStateHandler sets a function that is called on PortAudio's processing
// thread when the WASAPI stream changes state, or removes it if h is nil. It is
// useful to notice that the device has been unplugged.
//
// Only callback streams have a processing thread, so h is never called for
// blocking streams; their ReadStream and WriteStream return an error instead.
// WASAPIStreamStateThreadStop may arrive after Stop or Abort has returned.
func (s *Stream) SetWASAPIStateHandler(h WASAPIStateHandler) error {
	if s.ptr == nil {
		return ErrBadStreamPtr
	}
	if paWasapiSetStreamStateHandler == nil {
		return errMissingSymbol("PaWasapi_SetStreamStateHandler")
	}
	s.wasapiCallback.Store(h)
	if h == nil {
		return newError(paWasapiSetStreamStateHandler(s.ptr, 0, 0))
	}
	wasapiStateCallbackOnce.Do(func() {
		wasapiStateCallbackPtr = purego.NewCallback(wasapiStateCallbackTrampoline)
	})
	return newError(paWasapiSetStreamStateHandler(s.ptr, wasapiStateCallbackPtr, s.id))
}

func wasapiStateCallbackTrampoline(stream unsafe.Pointer, stateFlags, errorID, userData uintptr) uintptr {
	if s := lookupStream(userData); s != nil {
		if h, _ := s.wasapiCallback.Load().(WASAPIStateHandler); h != nil {
			h(WASAPIStreamState(uint32(stateFlags)), uint32(errorID))
		}
	}
	return 0
}
