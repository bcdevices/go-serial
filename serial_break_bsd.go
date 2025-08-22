//go:build darwin || freebsd || openbsd

package serial

import (
	"time"

	"golang.org/x/sys/unix"
)

// Break sends/controls a BREAK condition on macOS/*BSD.
//
// Semantics:
//
//	d > 0:
//	    No kernel-timed duration exists on these OSes. We assert BREAK with
//	    TIOCSBRK, sleep in userspace for d, then clear with TIOCCBRK.
//	    (On CDC-ACM this corresponds to SEND_BREAK wValue=0xFFFF, then later 0x0000.)
//	d == BreakDefault (0):
//	    Uses tcsendbreak(fd, 0) → POSIX default (~250 ms). Note: duration
//	    argument is ignored by these OSes, only the default is supported.
//	d == BreakIndefinite (-1):
//	    Assert indefinite BREAK via TIOCSBRK (CDC wValue=0xFFFF).
//	d == BreakStop (-2):
//	    Clear BREAK via TIOCCBRK (CDC wValue=0x0000).
//
// Notes:
//   - Timed breaks here are host-timed (userspace Sleep). Under load, actual
//     time may vary slightly.
//   - Some devices may not honor BREAK or may treat it as start/stop only.
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
		if err := unix.IoctlSetInt(fd, unix.TIOCSBRK, 0); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		return unix.IoctlSetInt(fd, unix.TIOCCBRK, 0)

	case d > 0:
		if err := unix.IoctlSetInt(fd, unix.TIOCSBRK, 0); err != nil {
			return err
		}
		time.Sleep(d)
		return unix.IoctlSetInt(fd, unix.TIOCCBRK, 0)

	default:
		return &PortError{code: InvalidTimeoutValue}
	}
}
