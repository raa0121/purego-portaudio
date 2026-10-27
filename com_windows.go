package portaudio

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modole32                = windows.NewLazySystemDLL("ole32.dll")
	procCoIncrementMTAUsage = modole32.NewProc("CoIncrementMTAUsage")
	procCoDecrementMTAUsage = modole32.NewProc("CoDecrementMTAUsage")
	procCoInitializeEx      = modole32.NewProc("CoInitializeEx")
	procCoUninitialize      = modole32.NewProc("CoUninitialize")

	mtaOnce   sync.Once
	mtaErr    error
	mtaCookie uintptr
)

// PortAudio's WASAPI and DirectSound backends require COM on every thread that
// calls them, but a goroutine may run each call on a different OS thread that
// never initialized COM. Keeping the process-wide multithreaded apartment alive
// makes every such thread an implicit MTA member.
func ensureMTA() error {
	mtaOnce.Do(func() {
		r, _, _ := procCoIncrementMTAUsage.Call(uintptr(unsafe.Pointer(&mtaCookie)))
		if hr := int32(r); hr < 0 {
			mtaErr = fmt.Errorf("portaudio: CoIncrementMTAUsage failed: HRESULT 0x%08x", uint32(hr))
		}
	})
	return mtaErr
}

// decrementMTA releases the process-wide MTA reference acquired by ensureMTA.
// It is safe to call when ensureMTA has not been called or has failed.
func decrementMTA() {
	if mtaCookie != 0 {
		procCoDecrementMTAUsage.Call(mtaCookie)
		mtaCookie = 0
	}
}

func initialize() (int32, error) {
	if err := ensureMTA(); err != nil {
		return 0, err
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Joining the MTA first makes PortAudio's own CoInitialize fail with
	// RPC_E_CHANGED_MODE, which it tolerates, instead of turning this Go thread
	// into a single-threaded apartment that needs a message loop.
	r, _, _ := procCoInitializeEx.Call(0, windows.COINIT_MULTITHREADED)
	if int32(r) >= 0 {
		defer procCoUninitialize.Call()
	}
	return paInitialize(), nil
}
