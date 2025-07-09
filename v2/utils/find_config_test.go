package utils_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"os"

	"github.com/discourse/launcher/v2/utils"
)

var _ = Describe("FindConfig", func() {

	It("Parses and returns yml or yaml files", func() {
		os.Setenv("COMP_LINE", "launcher build --conf-dir ../test/containers") //nolint:errcheck
		Expect(utils.FindConfigNames()).To(ContainElements("test", "test2"))
	})

	It("Parses and returns yml or yaml files with trailing slash", func() {
		os.Setenv("COMP_LINE", "launcher build --conf-dir ../test/containers/") //nolint:errcheck
		Expect(utils.FindConfigNames()).To(ContainElements("test", "test2"))
	})

	It("Parses and returns yml or yaml files on equals", func() {
		os.Setenv("COMP_LINE", "launcher --conf-dir=../test/containers other args") //nolint:errcheck
		Expect(utils.FindConfigNames()).To(ContainElements("test", "test2"))
	})

	It("doesn't error when dir does not exist when set", func() {
		os.Setenv("COMP_LINE", "launcher --conf-dir=./does-not-exist") //nolint:errcheck
		Expect(utils.FindConfigNames()).To(BeEmpty())
	})

	It("doesn't error when dir does not exist", func() {
		//by default it look is in ./containers directory, which does not exist
		// in this directory
		os.Setenv("COMP_LINE", "launcher") //nolint:errcheck
		Expect(utils.FindConfigNames()).To(BeEmpty())
	})
})
