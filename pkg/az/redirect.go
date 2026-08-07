package az

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// maxRedirectAttempts bounds how many ports an interactive login will try before
// giving up. Contention on the loopback ephemeral range is transient, so a small
// number of retries is enough; an unbounded loop would hang a login instead of
// reporting a problem the user can act on.
const maxRedirectAttempts = 5

// errPortInUse marks a bind failure as retryable. Anything else (a missing
// loopback interface, a sandbox that forbids listening) will fail identically on
// every port, so retrying it only delays the report.
var errPortInUse = errors.New("redirect port unavailable")

// reservePort binds an ephemeral loopback port and returns both the port and the
// listener still holding it. The listener is the reservation: asking the kernel
// for a free port and then closing it hands the port back, and any other process
// on the machine may take it before the caller binds. Callers must close the
// returned listener immediately before the real redirect listener binds, which
// keeps the unavoidable race down to that single handoff.
func reservePort() (int, net.Listener, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", errPortInUse, err)
	}
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		_ = l.Close()
		return 0, nil, fmt.Errorf("reserving a redirect port: unexpected address type %T", l.Addr())
	}
	return addr.Port, l, nil
}

// withRedirectPort reserves a loopback port, releases the reservation, and calls
// bind. A bind that fails because the port was taken in that window is retried
// on a fresh port, up to maxRedirectAttempts. The final error reports how many
// ports were tried so a genuinely contended machine is distinguishable from a
// configuration problem.
func withRedirectPort(ctx context.Context, bind func(port int) error) error {
	var lastErr error
	for attempt := 1; attempt <= maxRedirectAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		port, holder, err := reservePort()
		if err != nil {
			lastErr = err
			continue
		}
		// The reservation is released only once, immediately before bind, so a
		// bind failure below is a real race rather than self-contention.
		if cerr := holder.Close(); cerr != nil {
			lastErr = cerr
			continue
		}

		err = bind(port)
		if err == nil {
			return nil
		}
		if !isAddrInUse(err) {
			// A non-bind failure, such as a cancelled or rejected sign-in, is
			// the caller's answer and must not be retried on another port.
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("no redirect port available after %d attempts: %w", maxRedirectAttempts, lastErr)
}

// isAddrInUse reports whether err is a bind failure worth retrying on another
// port. MSAL wraps the listener error, so the check unwraps rather than
// comparing types at the top level.
func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errPortInUse) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Op == "listen"
	}
	return false
}
