//go:build windows

package serial

import (
	"time"

	"golang.org/x/sys/windows"
)

// Break sends/controls a BREAK condition on Windows.
//
// Semantics:
//
//	d > 0:
//	    No kernel-timed duration API exists. We assert BREAK with SetCommBreak,
//	    sleep in userspace for d, then clear with ClearCommBreak.
//	d == BreakDefault (0):
//	    Uses a ~250 ms default (assert → sleep → clear).
//	d == BreakIndefinite (-1):
//	    Assert indefinite BREAK via SetCommBreak.
//	d == BreakStop (-2):
//	    Clear BREAK via ClearCommBreak.
//
// Notes:
//   - Timing here is host-side (userspace Sleep).
//   - On CDC-ACM, this corresponds to SEND_BREAK wValue=0xFFFF (start) and
//     later wValue=0x0000 (stop) at the USB control level.
func (p *Port) Break(d time.Duration) error {
	if err := p.checkValid(); err != nil {
		return err
	}
	h := windows.Handle(p.internal.handle)

	switch {
	case d == BreakStop:
		return windows.ClearCommBreak(h)

	case d == BreakIndefinite:
		return windows.SetCommBreak(h)

	case d == BreakDefault, d > 0:
		if err := windows.SetCommBreak(h); err != nil {
			return err
		}
		dur := d
		if dur == BreakDefault {
			dur = 250 * time.Millisecond
		}
		time.Sleep(dur)
		return windows.ClearCommBreak(h)

	default:
		return &PortError{code: InvalidTimeoutValue}
	}
}
