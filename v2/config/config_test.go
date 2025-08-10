package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"errors"
	"os"
	"strings"

	"github.com/discourse/launcher/v2/config"
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
		err := conf.WriteYamlConfig(testDir, "config.yaml")
		Expect(err).To(BeNil())
		out, err := os.ReadFile(testDir + "/config.yaml")
		Expect(err).To(BeNil())
		Expect(strings.Contains(string(out[:]), ""))
		Expect(string(out[:])).To(ContainSubstring("DISCOURSE_DEVELOPER_EMAILS: 'me@example.com,you@example.com'"))
	})

	It("can convert pups config to dockerfile format and bake in default env", func() {
		dockerfile := conf.Dockerfile("", false, "config.yaml")
		Expect(dockerfile).To(ContainSubstring(`FROM ${dockerfile_from_image}
ARG LANG
ARG LANGUAGE
ARG LC_ALL
ARG MULTI
ARG RAILS_ENV
ARG REPLACED
ARG RUBY_GC_HEAP_GROWTH_MAX_SLOTS
ARG RUBY_GC_HEAP_INIT_SLOTS
ARG RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR
ARG UNICORN_SIDEKIQS
ARG UNICORN_WORKERS
ENV RAILS_ENV=${RAILS_ENV}
ENV RUBY_GC_HEAP_GROWTH_MAX_SLOTS=${RUBY_GC_HEAP_GROWTH_MAX_SLOTS}
ENV RUBY_GC_HEAP_INIT_SLOTS=${RUBY_GC_HEAP_INIT_SLOTS}
ENV RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=${RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR}
ENV UNICORN_SIDEKIQS=${UNICORN_SIDEKIQS}
ENV UNICORN_WORKERS=${UNICORN_WORKERS}
EXPOSE 443
EXPOSE 80
EXPOSE 90
COPY config.yaml /temp-config.yaml
RUN cat /temp-config.yaml | /usr/local/bin/pups  --stdin && rm /temp-config.yaml
CMD ["/sbin/boot"]`))
	})

	It("can generate a dockerfile with all env baked into the image", func() {
		dockerfile := conf.Dockerfile("", true, "config.yaml")
		Expect(dockerfile).To(ContainSubstring(`FROM ${dockerfile_from_image}
ARG LANG
ARG LANGUAGE
ARG LC_ALL
ARG MULTI
ARG RAILS_ENV
ARG REPLACED
ARG RUBY_GC_HEAP_GROWTH_MAX_SLOTS
ARG RUBY_GC_HEAP_INIT_SLOTS
ARG RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR
ARG UNICORN_SIDEKIQS
ARG UNICORN_WORKERS
ENV LANG=${LANG}
ENV LANGUAGE=${LANGUAGE}
ENV LC_ALL=${LC_ALL}
ENV MULTI=${MULTI}
ENV RAILS_ENV=${RAILS_ENV}
ENV REPLACED=${REPLACED}
ENV RUBY_GC_HEAP_GROWTH_MAX_SLOTS=${RUBY_GC_HEAP_GROWTH_MAX_SLOTS}
ENV RUBY_GC_HEAP_INIT_SLOTS=${RUBY_GC_HEAP_INIT_SLOTS}
ENV RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=${RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR}
ENV UNICORN_SIDEKIQS=${UNICORN_SIDEKIQS}
ENV UNICORN_WORKERS=${UNICORN_WORKERS}
EXPOSE 443
EXPOSE 80
EXPOSE 90
COPY config.yaml /temp-config.yaml
RUN cat /temp-config.yaml | /usr/local/bin/pups  --stdin && rm /temp-config.yaml
CMD ["/sbin/boot"]`))
	})

	Context("hostname tests", func() {
		It("replaces hostname", func() {
			config := config.Config{Env: map[string]string{"DOCKER_USE_HOSTNAME": "true", "DISCOURSE_HOSTNAME": "asdfASDF"}}
			Expect(config.GetDockerHostname("")).To(Equal("asdfASDF"))
		})
		It("replaces hostname", func() {
			config := config.Config{Env: map[string]string{"DOCKER_USE_HOSTNAME": "true", "DISCOURSE_HOSTNAME": "asdf!@#$%^&*()ASDF"}}
			Expect(config.GetDockerHostname("")).To(Equal("asdf----------ASDF"))
		})
		It("replaces a default hostnamehostname", func() {
			config := config.Config{}
			Expect(config.GetDockerHostname("asdf!@#")).To(Equal("asdf---"))
		})
	})

	It("should error if no base config LoadConfig to load yaml configuration", func() {
		_, err := config.LoadConfig("../test/containers", "test-no-base-image", true, "../test")
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("no base image specified in config, set base image with `base_image: {imagename}`"))
	})

	It("should be able to run LoadConfig to load yaml configuration", func() {
		conf, err := config.LoadConfig("../test/containers", "test-incompatible-plugin", true, "../test")
		Expect(err).To(BeNil())
		Expect(conf.ValidateConfig(errors.New("test"))).To(MatchError("test: the plugin 'discourse-reactions' is bundled with Discourse"))
	})
	It("should find the correct base image", func() {
		conf, err := config.LoadConfig("../test/containers", "test4-base-image-override", true, "../test")
		Expect(err).To(BeNil())
		Expect(conf.BaseImage).To(Equal("test"))
	})


	It("can configure build mounts", func() {
		conf, _ = config.LoadConfig("../test/containers", "test-mounts", true, "../test")
		dockerfile := conf.Dockerfile("", false, "config.yaml")
		Expect(dockerfile).To(ContainSubstring(`FROM ${dockerfile_from_image}
ARG LANG
ARG LANGUAGE
ARG LC_ALL
ARG MULTI
ARG RAILS_ENV
ARG REPLACED
ARG RUBY_GC_HEAP_GROWTH_MAX_SLOTS
ARG RUBY_GC_HEAP_INIT_SLOTS
ARG RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR
ARG UNICORN_SIDEKIQS
ARG UNICORN_WORKERS
ENV RAILS_ENV=${RAILS_ENV}
ENV RUBY_GC_HEAP_GROWTH_MAX_SLOTS=${RUBY_GC_HEAP_GROWTH_MAX_SLOTS}
ENV RUBY_GC_HEAP_INIT_SLOTS=${RUBY_GC_HEAP_INIT_SLOTS}
ENV RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=${RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR}
ENV UNICORN_SIDEKIQS=${UNICORN_SIDEKIQS}
ENV UNICORN_WORKERS=${UNICORN_WORKERS}
EXPOSE 443
EXPOSE 80
EXPOSE 90
COPY config.yaml /temp-config.yaml
RUN --mount=type=cache,target=/var/test/cache --mount=type=cache,target=/tmp/cache2 cat /temp-config.yaml | /usr/local/bin/pups  --stdin && rm /temp-config.yaml
CMD ["/sbin/boot"]`))
	})
})
