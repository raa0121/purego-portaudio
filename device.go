package portaudio

import (
	"fmt"
	"time"
)

// HostAPIIndex is an index into the list of host APIs, from 0 to HostAPICount()-1.
type HostAPIIndex int

// DeviceIndex is an index into the list of devices, from 0 to DeviceCount()-1.
type DeviceIndex int

const (
	// NoDevice denotes a missing device.
	NoDevice DeviceIndex = -1
	// UseHostAPISpecificDeviceSpecification indicates that the device is given in
	// StreamParameters.HostAPISpecificStreamInfo.
	UseHostAPISpecificDeviceSpecification DeviceIndex = -2
)

// HostAPITypeID identifies a host API implementation (PaHostApiTypeId).
type HostAPITypeID int

const (
	InDevelopment   HostAPITypeID = 0
	DirectSound     HostAPITypeID = 1
	MME             HostAPITypeID = 2
	ASIO            HostAPITypeID = 3
	SoundManager    HostAPITypeID = 4
	CoreAudio       HostAPITypeID = 5
	OSS             HostAPITypeID = 7
	ALSA            HostAPITypeID = 8
	AL              HostAPITypeID = 9
	BeOS            HostAPITypeID = 10
	WDMKS           HostAPITypeID = 11
	JACK            HostAPITypeID = 12
	WASAPI          HostAPITypeID = 13
	AudioScienceHPI HostAPITypeID = 14
	AudioIO         HostAPITypeID = 15
	PulseAudio      HostAPITypeID = 16
	Sndio           HostAPITypeID = 17
)

var hostAPITypeNames = map[HostAPITypeID]string{
	InDevelopment:   "InDevelopment",
	DirectSound:     "DirectSound",
	MME:             "MME",
	ASIO:            "ASIO",
	SoundManager:    "SoundManager",
	CoreAudio:       "CoreAudio",
	OSS:             "OSS",
	ALSA:            "ALSA",
	AL:              "AL",
	BeOS:            "BeOS",
	WDMKS:           "WDMKS",
	JACK:            "JACK",
	WASAPI:          "WASAPI",
	AudioScienceHPI: "AudioScienceHPI",
	AudioIO:         "AudioIO",
	PulseAudio:      "PulseAudio",
	Sndio:           "Sndio",
}

func (t HostAPITypeID) String() string {
	if name, ok := hostAPITypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("HostAPITypeID(%d)", int(t))
}

// HostAPIInfo describes a host API.
type HostAPIInfo struct {
	Index HostAPIIndex
	Type  HostAPITypeID
	Name  string
	// DeviceCount is the number of devices belonging to this host API. Use
	// DeviceIndexOf to convert a host API device index to a DeviceIndex.
	DeviceCount         int
	DefaultInputDevice  DeviceIndex
	DefaultOutputDevice DeviceIndex
}

type cHostAPIInfo struct {
	structVersion       int32
	typeID              int32
	name                *byte
	deviceCount         int32
	defaultInputDevice  int32
	defaultOutputDevice int32
}

// DeviceInfo describes a device.
type DeviceInfo struct {
	Index                    DeviceIndex
	Name                     string
	HostAPI                  HostAPIIndex
	MaxInputChannels         int
	MaxOutputChannels        int
	DefaultLowInputLatency   time.Duration
	DefaultLowOutputLatency  time.Duration
	DefaultHighInputLatency  time.Duration
	DefaultHighOutputLatency time.Duration
	DefaultSampleRate        float64
}

type cDeviceInfo struct {
	structVersion            int32
	name                     *byte
	hostAPI                  int32
	maxInputChannels         int32
	maxOutputChannels        int32
	defaultLowInputLatency   float64
	defaultLowOutputLatency  float64
	defaultHighInputLatency  float64
	defaultHighOutputLatency float64
	defaultSampleRate        float64
}

// HostAPICount returns the number of available host APIs.
func HostAPICount() (int, error) {
	if !loaded.Load() {
		return 0, ErrNotInitialized
	}
	n := paGetHostApiCount()
	if n < 0 {
		return 0, newError(n)
	}
	return int(n), nil
}

// DefaultHostAPI returns the index of the default host API.
func DefaultHostAPI() (HostAPIIndex, error) {
	if !loaded.Load() {
		return 0, ErrNotInitialized
	}
	i := paGetDefaultHostApi()
	if i < 0 {
		return 0, newError(i)
	}
	return HostAPIIndex(i), nil
}

// HostAPI returns information about the host API at index.
func HostAPI(index HostAPIIndex) (*HostAPIInfo, error) {
	if !loaded.Load() {
		return nil, ErrNotInitialized
	}
	p := paGetHostApiInfo(int32(index))
	if p == nil {
		return nil, ErrInvalidHostAPI
	}
	c := (*cHostAPIInfo)(p)
	return &HostAPIInfo{
		Index:               index,
		Type:                HostAPITypeID(c.typeID),
		Name:                goString(c.name),
		DeviceCount:         int(c.deviceCount),
		DefaultInputDevice:  DeviceIndex(c.defaultInputDevice),
		DefaultOutputDevice: DeviceIndex(c.defaultOutputDevice),
	}, nil
}

// HostAPIs returns information about all available host APIs.
func HostAPIs() ([]*HostAPIInfo, error) {
	n, err := HostAPICount()
	if err != nil {
		return nil, err
	}
	apis := make([]*HostAPIInfo, n)
	for i := range apis {
		if apis[i], err = HostAPI(HostAPIIndex(i)); err != nil {
			return nil, err
		}
	}
	return apis, nil
}

// HostAPIIndexOf returns the index of the host API of type t. It returns
// ErrHostAPINotFound if that host API is not available.
func HostAPIIndexOf(t HostAPITypeID) (HostAPIIndex, error) {
	if !loaded.Load() {
		return 0, ErrNotInitialized
	}
	i := paHostApiTypeIdToHostApiIndex(int32(t))
	if i < 0 {
		return 0, newError(i)
	}
	return HostAPIIndex(i), nil
}

// DeviceIndexOf converts a device index local to a host API, from 0 to
// HostAPIInfo.DeviceCount-1, to a DeviceIndex.
func DeviceIndexOf(hostAPI HostAPIIndex, hostAPIDeviceIndex int) (DeviceIndex, error) {
	if !loaded.Load() {
		return NoDevice, ErrNotInitialized
	}
	i := paHostApiDeviceIndexToDeviceIndex(int32(hostAPI), int32(hostAPIDeviceIndex))
	if i < 0 {
		return NoDevice, newError(i)
	}
	return DeviceIndex(i), nil
}

// DeviceCount returns the number of available devices.
func DeviceCount() (int, error) {
	if !loaded.Load() {
		return 0, ErrNotInitialized
	}
	n := paGetDeviceCount()
	if n < 0 {
		return 0, newError(n)
	}
	return int(n), nil
}

// DefaultInputDevice returns the default input device, or NoDevice if there is
// none or PortAudio is not initialized.
func DefaultInputDevice() DeviceIndex {
	if !loaded.Load() {
		return NoDevice
	}
	return DeviceIndex(paGetDefaultInputDevice())
}

// DefaultOutputDevice returns the default output device, or NoDevice if there is
// none or PortAudio is not initialized.
func DefaultOutputDevice() DeviceIndex {
	if !loaded.Load() {
		return NoDevice
	}
	return DeviceIndex(paGetDefaultOutputDevice())
}

// Device returns information about the device at index.
func Device(index DeviceIndex) (*DeviceInfo, error) {
	if !loaded.Load() {
		return nil, ErrNotInitialized
	}
	p := paGetDeviceInfo(int32(index))
	if p == nil {
		return nil, ErrInvalidDevice
	}
	c := (*cDeviceInfo)(p)
	return &DeviceInfo{
		Index:                    index,
		Name:                     goString(c.name),
		HostAPI:                  HostAPIIndex(c.hostAPI),
		MaxInputChannels:         int(c.maxInputChannels),
		MaxOutputChannels:        int(c.maxOutputChannels),
		DefaultLowInputLatency:   secondsToDuration(c.defaultLowInputLatency),
		DefaultLowOutputLatency:  secondsToDuration(c.defaultLowOutputLatency),
		DefaultHighInputLatency:  secondsToDuration(c.defaultHighInputLatency),
		DefaultHighOutputLatency: secondsToDuration(c.defaultHighOutputLatency),
		DefaultSampleRate:        c.defaultSampleRate,
	}, nil
}

// Devices returns information about all available devices.
func Devices() ([]*DeviceInfo, error) {
	n, err := DeviceCount()
	if err != nil {
		return nil, err
	}
	devices := make([]*DeviceInfo, n)
	for i := range devices {
		if devices[i], err = Device(DeviceIndex(i)); err != nil {
			return nil, err
		}
	}
	return devices, nil
}
