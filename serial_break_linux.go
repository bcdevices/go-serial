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
//	    Uses tcsendbreak(fd, ms). The kernel cdc_acm driver sends ONE USB CDC
//	    SEND_BREAK with wValue=ms and the *device* times the break.
//	    Valid range 1..65534 ms (0xFFFE). Longer durations fall back to
//	    host-timed assert/sleep/clear.
//	d == BreakDefault (0):
//	    POSIX default break (~250 ms): tcsendbreak(fd, 0).
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
		return unix.Tcsendbreak(fd, 0)

	case d > 0:
		ms := int(d / time.Millisecond)
		if ms <= 0 {
			ms = 1
		}
		if ms <= 65534 {
			// Kernel issues a single SEND_BREAK (wValue=ms); device times it.
			return unix.Tcsendbreak(fd, ms)
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
