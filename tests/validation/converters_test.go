package validation_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rancher/wins/pkg/converters"
)

var _ = Describe("converters", func() {
	It("string to bytes", func() {
		expected := []byte("hello world")
		actual := converters.UnsafeStringToBytes("hello world")

		Expect(expected).Should(Equal(actual))
	})

	It("bytes to string", func() {
		expected := "hello world"
		actual := converters.UnsafeBytesToString([]byte("hello world"))

		Expect(expected).Should(Equal(actual))
	})

	It("to json", func() {
		testingObj := struct {
			A string
			B string `json:"bAlias"`
			C string `json:"-"`
		}{
			A: "a",
			B: "b",
			C: "c",
		}

		expected := `{"A":"a","bAlias":"b"}`

		actual, err := converters.ToJson(testingObj)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(expected).Should(Equal(actual))
	})
})
