package portaudio

import (
	"fmt"
	"unsafe"
)

// ErrorCode is a PortAudio error code (PaErrorCode). It implements error, so the
// constants below can be compared with errors.Is.
type ErrorCode int

const (
	ErrNotInitialized                        ErrorCode = -10000
	ErrUnanticipatedHostError                ErrorCode = -9999
	ErrInvalidChannelCount                   ErrorCode = -9998
	ErrInvalidSampleRate                     ErrorCode = -9997
	ErrInvalidDevice                         ErrorCode = -9996
	ErrInvalidFlag                           ErrorCode = -9995
	ErrSampleFormatNotSupported              ErrorCode = -9994
	ErrBadIODeviceCombination                ErrorCode = -9993
	ErrInsufficientMemory                    ErrorCode = -9992
	ErrBufferTooBig                          ErrorCode = -9991
	ErrBufferTooSmall                        ErrorCode = -9990
	ErrNullCallback                          ErrorCode = -9989
	ErrBadStreamPtr                          ErrorCode = -9988
	ErrTimedOut                              ErrorCode = -9987
	ErrInternalError                         ErrorCode = -9986
	ErrDeviceUnavailable                     ErrorCode = -9985
	ErrIncompatibleHostAPISpecificStreamInfo ErrorCode = -9984
	ErrStreamIsStopped                       ErrorCode = -9983
	ErrStreamIsNotStopped                    ErrorCode = -9982
	// ErrInputOverflowed is returned by ReadStream when input data was discarded
	// before the read. The buffer is still filled.
	ErrInputOverflowed ErrorCode = -9981
	// ErrOutputUnderflowed is returned by WriteStream when output data was not
	// supplied in time. The buffer has still been written.
	ErrOutputUnderflowed                ErrorCode = -9980
	ErrHostAPINotFound                  ErrorCode = -9979
	ErrInvalidHostAPI                   ErrorCode = -9978
	ErrCanNotReadFromACallbackStream    ErrorCode = -9977
	ErrCanNotWriteToACallbackStream     ErrorCode = -9976
	ErrCanNotReadFromAnOutputOnlyStream ErrorCode = -9975
	ErrCanNotWriteToAnInputOnlyStream   ErrorCode = -9974
	ErrIncompatibleStreamHostAPI        ErrorCode = -9973
	ErrBadBufferPtr                     ErrorCode = -9972
	ErrCanNotInitializeRecursively      ErrorCode = -9971
)

func (e ErrorCode) Error() string {
	if loaded.Load() {
		return "portaudio: " + paGetErrorText(int32(e))
	}
	return fmt.Sprintf("portaudio: error %d", int(e))
}

// HostError is returned in place of ErrUnanticipatedHostError and carries the
// error reported by the underlying host API.
type HostError struct {
	HostAPIType HostAPITypeID
	Code        int
	Text        string
}

func (e *HostError) Error() string {
	return fmt.Sprintf("portaudio: host error (%v): %s (code %d)", e.HostAPIType, e.Text, e.Code)
}

// Unwrap returns ErrUnanticipatedHostError.
func (e *HostError) Unwrap() error {
	return ErrUnanticipatedHostError
}

type cHostErrorInfo struct {
	hostAPIType int32
	errorCode   clong
	errorText   *byte
}

func newError(code int32) error {
	if code >= 0 {
		return nil
	}
	if ErrorCode(code) == ErrUnanticipatedHostError {
		// hostApiType stays -1 when the host API did not record any details.
		if p := paGetLastHostErrorInfo(); p != nil && (*cHostErrorInfo)(p).hostAPIType >= 0 {
			info := (*cHostErrorInfo)(p)
			return &HostError{
				HostAPIType: HostAPITypeID(info.hostAPIType),
				Code:        int(info.errorCode),
				Text:        goString(info.errorText),
			}
		}
	}
	return ErrorCode(code)
}

func goString(p *byte) string {
	if p == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(p, n))
}
