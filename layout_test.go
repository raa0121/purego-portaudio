package portaudio

import (
	"runtime"
	"testing"
	"unsafe"
)

// The Go mirrors must match the C struct layouts exactly; purego reads the
// shared library memory with these offsets.
func TestStructLayout(t *testing.T) {
	longSize := uintptr(8)
	if runtime.GOOS == "windows" {
		longSize = 4
	}

	tests := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"sizeof(PaVersionInfo)", unsafe.Sizeof(cVersionInfo{}), 32},
		{"offsetof(PaVersionInfo.versionText)", unsafe.Offsetof(cVersionInfo{}.versionText), 24},

		{"sizeof(PaHostApiInfo)", unsafe.Sizeof(cHostAPIInfo{}), 32},
		{"offsetof(PaHostApiInfo.name)", unsafe.Offsetof(cHostAPIInfo{}.name), 8},
		{"offsetof(PaHostApiInfo.defaultOutputDevice)", unsafe.Offsetof(cHostAPIInfo{}.defaultOutputDevice), 24},

		{"offsetof(PaHostErrorInfo.errorCode)", unsafe.Offsetof(cHostErrorInfo{}.errorCode), longSize},
		{"offsetof(PaHostErrorInfo.errorText)", unsafe.Offsetof(cHostErrorInfo{}.errorText), 2 * longSize},

		{"sizeof(PaDeviceInfo)", unsafe.Sizeof(cDeviceInfo{}), 72},
		{"offsetof(PaDeviceInfo.hostApi)", unsafe.Offsetof(cDeviceInfo{}.hostAPI), 16},
		{"offsetof(PaDeviceInfo.defaultLowInputLatency)", unsafe.Offsetof(cDeviceInfo{}.defaultLowInputLatency), 32},
		{"offsetof(PaDeviceInfo.defaultSampleRate)", unsafe.Offsetof(cDeviceInfo{}.defaultSampleRate), 64},

		{"sizeof(PaStreamParameters)", unsafe.Sizeof(cStreamParameters{}), 32},
		{"offsetof(PaStreamParameters.sampleFormat)", unsafe.Offsetof(cStreamParameters{}.sampleFormat), 8},
		{"offsetof(PaStreamParameters.suggestedLatency)", unsafe.Offsetof(cStreamParameters{}.suggestedLatency), 16},
		{"offsetof(PaStreamParameters.hostApiSpecificStreamInfo)", unsafe.Offsetof(cStreamParameters{}.hostAPISpecificStreamInfo), 24},

		{"sizeof(PaStreamInfo)", unsafe.Sizeof(cStreamInfo{}), 32},
		{"offsetof(PaStreamInfo.inputLatency)", unsafe.Offsetof(cStreamInfo{}.inputLatency), 8},

		{"sizeof(PaStreamCallbackTimeInfo)", unsafe.Sizeof(StreamCallbackTimeInfo{}), 24},
	}
	tests = append(tests, platformLayoutTests()...)

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}
