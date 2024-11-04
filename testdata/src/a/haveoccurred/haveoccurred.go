package haveoccurred

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HaveOccurred", func() {
	It("HaveOccurred for non error types", func() {
		err := fmt.Errorf("err")
		Expect(err).To(HaveOccurred())
	})
	It("HaveOccurred for non error types", func() {
		var p *int
		Expect(p).To(HaveOccurred()) // want `ginkgo-linter: asserting a non-error type with HaveOccurred matcher`
	})

	It("HaveOccurred for non error types", func() {
		Expect(fmt.Errorf("err")).To(HaveOccurred())
	})

})

var _ = Describe("Succeed", func() {
	It("valid: Succeed for error type", func() {
		err := fmt.Errorf("err")
		Expect(err).To(Succeed())
	})

	It("valid: Succeed for error func", func() {
		Expect(retOneErr()).To(Succeed())
	})

	It("Succeed for non error types", func() {
		var p *int
		Expect(p).To(Succeed()) // want `ginkgo-linter: asserting a non-error type with Succeed matcher`
	})

	It("Succeed for non error func", func() {
		Expect(retNonErr()).To(Succeed()) // want `ginkgo-linter: asserting a non-error type with Succeed matcher`
	})

	It("Succeed for muli-value + error func", func() {
		Expect(retValAndErr()).To(Succeed()) // want `ginkgo-linter: the Success matcher does not support multiple values`
	})

})