package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/discourse/launcher/v2/config"
	"os"
	"strings"
)

var _ = Describe("Config", func() {
	var testDir string
	var conf *config.Config
	BeforeEach(func() {
		testDir, _ = os.MkdirTemp("", "ddocker-test")
		conf, _ = config.LoadConfig("../test/containers", "test", true, "../test")
	})
	AfterEach(func() {
		os.RemoveAll(testDir) //nolint:errcheck
	})
	It("should be able to run LoadConfig to load yaml configuration", func() {
		conf, err := config.LoadConfig("../test/containers", "test", true, "../test")
		Expect(err).To(BeNil())
		result := conf.Yaml()
		Expect(result).To(ContainSubstring("DISCOURSE_DEVELOPER_EMAILS: 'me@example.com,you@example.com'"))
		Expect(result).To(ContainSubstring("_FILE_SEPERATOR_"))
		Expect(result).To(ContainSubstring("version: tests-passed"))
	})

	It("can write raw yaml config", func() {
		err := conf.WriteYamlConfig(testDir)
		Expect(err).To(BeNil())
		out, err := os.ReadFile(testDir + "/config.yaml")
		Expect(err).To(BeNil())
		Expect(strings.Contains(string(out[:]), ""))
		Expect(string(out[:])).To(ContainSubstring("DISCOURSE_DEVELOPER_EMAILS: 'me@example.com,you@example.com'"))
	})

	It("can convert pups config to dockerfile format", func() {
		dockerfile := conf.Dockerfile("", false)
		Expect(dockerfile).To(ContainSubstring("ARG DISCOURSE_DEVELOPER_EMAILS"))
		Expect(dockerfile).To(ContainSubstring("RUN cat /temp-config.yaml"))
		Expect(dockerfile).To(ContainSubstring("EXPOSE 80"))
	})

	Context("hostname tests", func() {
		It("replaces hostname", func() {
			config := config.Config{Env: map[string]string{"DOCKER_USE_HOSTNAME": "true", "DISCOURSE_HOSTNAME": "asdfASDF"}}
			Expect(config.DockerHostname("")).To(Equal("asdfASDF"))
		})
		It("replaces hostname", func() {
			config := config.Config{Env: map[string]string{"DOCKER_USE_HOSTNAME": "true", "DISCOURSE_HOSTNAME": "asdf!@#$%^&*()ASDF"}}
			Expect(config.DockerHostname("")).To(Equal("asdf----------ASDF"))
		})
		It("replaces a default hostnamehostname", func() {
			config := config.Config{}
			Expect(config.DockerHostname("asdf!@#")).To(Equal("asdf---"))
		})
	})
	It("should error if no base config LoadConfig to load yaml configuration", func() {
		_, err := config.LoadConfig("../test/containers", "test-no-base-image", true, "../test")
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("no base image specified in config! set base image with `base_image: {imagename}`"))
	})
})
