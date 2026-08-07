package az

import (
	"context"
	"errors"
	"fmt"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Redirect listener", func() {
	It("hands out a port that is free at the moment of the handoff", func() {
		var got int
		Expect(withRedirectPort(context.Background(), func(port int) error {
			got = port
			l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				return err
			}
			return l.Close()
		})).To(Succeed())
		Expect(got).To(BeNumerically(">", 0))
	})

	It("holds the port until the reservation is released", func() {
		port, holder, err := reservePort()
		Expect(err).NotTo(HaveOccurred())
		defer holder.Close()

		// While the reservation is held nobody else can take the port, which is
		// the whole point of returning the listener rather than just the number.
		_, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		Expect(err).To(HaveOccurred())
	})

	It("retries on another port after a bind failure", func() {
		var ports []int
		attempts := 0
		Expect(withRedirectPort(context.Background(), func(port int) error {
			ports = append(ports, port)
			attempts++
			if attempts < 3 {
				return &net.OpError{Op: "listen", Err: errors.New("address already in use")}
			}
			return nil
		})).To(Succeed())
		Expect(attempts).To(Equal(3))
		Expect(ports).To(HaveLen(3))
	})

	It("reports the attempt count once the retries are exhausted", func() {
		attempts := 0
		err := withRedirectPort(context.Background(), func(port int) error {
			attempts++
			return &net.OpError{Op: "listen", Err: errors.New("address already in use")}
		})
		Expect(err).To(HaveOccurred())
		Expect(attempts).To(Equal(maxRedirectAttempts))
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%d attempts", maxRedirectAttempts)))
	})

	It("does not retry a failure that is not a bind failure", func() {
		sentinel := errors.New("the user declined the sign-in")
		attempts := 0
		err := withRedirectPort(context.Background(), func(port int) error {
			attempts++
			return sentinel
		})
		Expect(errors.Is(err, sentinel)).To(BeTrue())
		Expect(attempts).To(Equal(1))
	})

	It("does not open a listener once the context is cancelled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		err := withRedirectPort(ctx, func(port int) error {
			called = true
			return nil
		})
		Expect(errors.Is(err, context.Canceled)).To(BeTrue())
		Expect(called).To(BeFalse())
	})
})
