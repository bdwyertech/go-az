package az

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAz(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/az Suite")
}
