//go:build !windows

package portaudio

import (
	"runtime"

	"github.com/ebitengine/purego"
)

func defaultLibraryNames() []string {
	if runtime.GOOS == "darwin" {
		return []string{
			"libportaudio.2.dylib",
			"libportaudio.dylib",
			"/opt/homebrew/lib/libportaudio.2.dylib",
			"/usr/local/lib/libportaudio.2.dylib",
		}
	}
	return []string{
		"libportaudio.so.2",
		"libportaudio.so",
	}
}

func openLibrary(name string) (uintptr, error) {
	return purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

func closeLibrary(handle uintptr) {
	_ = purego.Dlclose(handle)
}

func lookupSymbol(handle uintptr, name string) (uintptr, error) {
	return purego.Dlsym(handle, name)
}
