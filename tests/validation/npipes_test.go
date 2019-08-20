package validation_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rancher/wins/pkg/npipes"
	"github.com/rancher/wins/pkg/powershells"
)

var _ = Describe("npipes", func() {
	It("pipe creation", func() {
		// invalid path
		_, err := npipes.New("//./pipe/test", "", 0)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).Should(HavePrefix("could not recognize path:"))

		l, err := npipes.New(npipes.GetFullPath("test"), "", 0)
		Expect(err).NotTo(HaveOccurred())

		// duplicate listener
		_, err = npipes.New("npipe:////./pipe/test", "", 0)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).Should(HaveSuffix("Access is denied."))

		l.Close()
	})

	It("new a pipe", func() {
		var psOutput interface{}
		commandOut := func(output interface{}) {
			psOutput = output
			consoleInfo(output)
		}

		psb := &powershells.Builder{}
		ps, err := psb.StdOut(commandOut).StdErr(consoleError).Build()
		if err != nil {
			consoleFatal(err)
		}

		npipeName := `rancher_wins`
		l, err := npipes.New(npipes.GetFullPath(npipeName), "", 0)
		Expect(err).NotTo(HaveOccurred())

		// check this npipe via PowerShell
		err = ps.ExecuteCommand(context.TODO(), fmt.Sprintf(`Get-ChildItem \\.\pipe\ -ErrorAction Ignore | ? Name -eq %s | Select-Object -ExpandProperty Name | Write-Host -NoNewline`, npipeName))
		Expect(err).NotTo(HaveOccurred())
		Expect(psOutput).To(Equal(npipeName))

		psOutput = ""
		l.Close()
		err = ps.ExecuteCommand(context.TODO(), fmt.Sprintf(`Get-ChildItem \\.\pipe\ -ErrorAction Ignore | ? Name -eq %s | Select-Object -ExpandProperty Name | Write-Host -NoNewline`, npipeName))
		Expect(err).NotTo(HaveOccurred())
		Expect(psOutput).To(Equal(""))
	})
})
