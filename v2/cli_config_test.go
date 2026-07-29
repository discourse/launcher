package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"
	"context"

	ddocker "github.com/discourse/launcher/v2"
	"github.com/discourse/launcher/v2/utils"
)

var _ = Describe("Resolve", func() {
	var out *bytes.Buffer
	var cli *ddocker.Cli
	var ctx context.Context

	BeforeEach(func() {
		out = &bytes.Buffer{}
		utils.Out = out
		ctx = context.Background()
		cli = &ddocker.Cli{
			ConfDir:      "./test/containers",
			TemplatesDir: "./test",
		}
	})

	It("prints the full resolved yaml when no template is given", func() {
		resolve := ddocker.ResolveCmd{Config: "test"}
		err := resolve.Run(cli, ctx)
		Expect(err).To(BeNil())
		Expect(out.String()).To(ContainSubstring("base_image: discourse/base:2.0.20250226-0128"))
		Expect(out.String()).To(ContainSubstring("version: tests-passed"))
	})

	It("renders a template referencing yaml keys", func() {
		resolve := ddocker.ResolveCmd{Config: "test", Template: "{{.base_image}}"}
		err := resolve.Run(cli, ctx)
		Expect(err).To(BeNil())
		Expect(out.String()).To(Equal("discourse/base:2.0.20250226-0128\n"))
	})

	It("applies --set overrides before rendering", func() {
		resolve := ddocker.ResolveCmd{
			Config:          "test",
			Template:        "{{.base_image}}",
			ConfigOverrides: map[string]string{"base_image": "my/override:1"},
		}
		err := resolve.Run(cli, ctx)
		Expect(err).To(BeNil())
		Expect(out.String()).To(Equal("my/override:1\n"))
	})
})
