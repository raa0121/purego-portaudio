package portaudio

import (
	"sync/atomic"
	"time"
)

// initializeCount tracks successful Initialize calls so that the last matching
// Terminate can release Windows COM resources.
var initializeCount atomic.Int32

// VersionInfo describes the version of the loaded PortAudio library.
type VersionInfo struct {
	Major, Minor, SubMinor int
	// ControlRevision is the Git revision PortAudio was built from, if known.
	ControlRevision string
	Text            string
}

type cVersionInfo struct {
	versionMajor           int32
	versionMinor           int32
	versionSubMinor        int32
	versionControlRevision *byte
	versionText            *byte
}

// Version returns the version of the PortAudio library, loading it with [Load] if
// needed. Initialize does not have to be called first.
func Version() (*VersionInfo, error) {
	if err := Load(); err != nil {
		return nil, err
	}
	p := paGetVersionInfo()
	if p == nil {
		return nil, ErrInternalError
	}
	c := (*cVersionInfo)(p)
	return &VersionInfo{
		Major:           int(c.versionMajor),
		Minor:           int(c.versionMinor),
		SubMinor:        int(c.versionSubMinor),
		ControlRevision: goString(c.versionControlRevision),
		Text:            goString(c.versionText),
	}, nil
}

// Initialize loads the PortAudio library with [Load] if needed and initializes it.
// Each successful call must be matched by a call to [Terminate].
//
// On Windows, Initialize keeps the process-wide COM multithreaded apartment
// alive so that PortAudio can be called from any goroutine. Host APIs that
// need a single-threaded apartment, such as ASIO, are therefore not supported.
func Initialize() error {
	if err := Load(); err != nil {
		return err
	}
	r, err := initialize()
	if err != nil {
		return err
	}
	if err := newError(r); err != nil {
		return err
	}
	initializeCount.Add(1)
	return nil
}

// Terminate deinitializes PortAudio, closing any streams that are still open.
func Terminate() error {
	if !loaded.Load() {
		return ErrNotInitialized
	}
	count := initializeCount.Add(-1)
	if count < 0 {
		initializeCount.Add(1)
		return ErrNotInitialized
	}
	if count > 0 {
		return newError(paTerminate())
	}
	err := newError(paTerminate())
	decrementMTA()
	return err
}

// SampleSize returns the size in bytes of a single sample of format.
func SampleSize(format SampleFormat) (int, error) {
	if !loaded.Load() {
		return 0, ErrNotInitialized
	}
	n := paGetSampleSize(culong(format))
	if n < 0 {
		return 0, newError(n)
	}
	return int(n), nil
}

// Sleep sleeps for at least d, with millisecond resolution, using PortAudio's
// portable sleep.
func Sleep(d time.Duration) {
	if !loaded.Load() {
		time.Sleep(d)
		return
	}
	paSleep(clong(d.Milliseconds()))
}

func secondsToDuration(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}
