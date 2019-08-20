package validation_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation Suite")
}

func consoleInfo(msg interface{}) {
	fmt.Fprintf(GinkgoWriter, ">>Info>> %v\n", msg)
}

func consoleWarn(msg interface{}) {
	fmt.Fprintf(GinkgoWriter, ">>Warn>> %v\n", msg)
}

func consoleError(msg interface{}) {
	fmt.Fprintf(GinkgoWriter, ">>Erro>> %v\n", msg)
}

func consoleFatal(msg interface{}) {
	Fail(fmt.Sprintf(">>Fata>> %v\n", msg))
}
