package portaudio

import "unsafe"

func platformLayoutTests() []struct {
	name string
	got  uintptr
	want uintptr
} {
	return []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"sizeof(PaWasapiStreamInfo)", unsafe.Sizeof(cWASAPIStreamInfo{}), 72},
		{"offsetof(PaWasapiStreamInfo.hostProcessorOutput)", unsafe.Offsetof(cWASAPIStreamInfo{}.hostProcessorOutput), 24},
		{"offsetof(PaWasapiStreamInfo.threadPriority)", unsafe.Offsetof(cWASAPIStreamInfo{}.threadPriority), 40},
		{"offsetof(PaWasapiStreamInfo.passthrough)", unsafe.Offsetof(cWASAPIStreamInfo{}.passthrough), 52},
	}
}
