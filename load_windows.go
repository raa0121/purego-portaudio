package portaudio

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

func defaultLibraryNames() []string {
	return []string{
		"portaudio.dll",
		"libportaudio.dll",
		"libportaudio-2.dll",
		"portaudio_x64.dll",
	}
}

// The current directory and PATH are excluded from the search to avoid DLL
// preloading attacks. Pass an absolute path to Load for other locations.
func openLibrary(name string) (uintptr, error) {
	flags := uintptr(windows.LOAD_LIBRARY_SEARCH_DEFAULT_DIRS)
	if filepath.IsAbs(name) {
		// Lets a MinGW build find libgcc / libwinpthread next to the DLL.
		flags |= windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR
	}
	h, err := windows.LoadLibraryEx(name, 0, flags)
	return uintptr(h), err
}

func closeLibrary(handle uintptr) {
	_ = windows.FreeLibrary(windows.Handle(handle))
}

func lookupSymbol(handle uintptr, name string) (uintptr, error) {
	return windows.GetProcAddress(windows.Handle(handle), name)
}
