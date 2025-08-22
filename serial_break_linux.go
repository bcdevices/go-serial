//go:build linux

package serial

import (
	"time"

	"golang.org/x/sys/unix"
)

// Break sends a BREAK condition on the serial line (Linux).
//
// Semantics:
//
//	d > 0:
//	    Timed break via TCSBRKP with duration rounded to deciseconds
//	    Uses tcsendbreak(fd, ms). The kernel cdc_acm driver sends ONE USB CDC
//	    SEND_BREAK with wValue=ms and the *device* times the break.
//	    Valid range 1..65534 ms (0xFFFE). Longer durations fall back to
//	    host-timed assert/sleep/clear.
//	d == BreakDefault (0):
//	    POSIX default break (~250 ms) via TCSBRKP with 0
//	d == BreakIndefinite (-1):
//	    Assert indefinite break: TIOCSBRK (CDC wValue=0xFFFF).
//	d == BreakStop (-2):
//	    Clear break: TIOCCBRK (CDC wValue=0x0000).
func (p *Port) Break(d time.Duration) error {
	if err := p.checkValid(); err != nil {
		return err
	}
	fd := p.internal.handle

	switch {
	case d == BreakStop:
		return unix.IoctlSetInt(fd, unix.TIOCCBRK, 0)

	case d == BreakIndefinite:
		return unix.IoctlSetInt(fd, unix.TIOCSBRK, 0)

	case d == BreakDefault:
		return unix.IoctlSetInt(fd, unix.TCSBRKP, 0)

	case d > 0:
		ms := int(d / time.Millisecond)
		if ms <= 0 {
			ms = 1
		}
		if ms <= 65534 {
			// Round to deciseconds (0.1 s units). 0 means default; use >=1.
			deci := (ms + 50) / 100
			if deci < 1 {
				deci = 1
			}
			return unix.IoctlSetInt(fd, unix.TCSBRKP, deci)
		}

		// Fallback for durations >65.534s: host-timed start/sleep/stop.
		if err := unix.IoctlSetInt(fd, unix.TIOCSBRK, 0); err != nil {
			return err
		}
		time.Sleep(d)
		return unix.IoctlSetInt(fd, unix.TIOCCBRK, 0)

	default:
		return &PortError{code: InvalidTimeoutValue}
	}
}
