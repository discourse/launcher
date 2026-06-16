package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"errors"
	"os"

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
		Expect(string(out[:])).To(ContainSubstring("DISCOURSE_DEVELOPER_EMAILS: 'me@example.com,you@example.com'"))
	})

	It("appends {{config}} replaced env values to the raw yaml config", func() {
		err := conf.WriteYamlConfig(testDir, "config.yaml")
		Expect(err).To(BeNil())
		out, err := os.ReadFile(testDir + "/config.yaml")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring("REPLACED: test/test/test"))
	})

	It("can convert pups config to dockerfile format and bake in default env", func() {
		dockerfile := conf.Dockerfile(false, false, "config.yaml")
		Expect(dockerfile).To(ContainSubstring(`FROM ${dockerfile_from_image} AS discourse-full
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
ENV RAILS_ENV=${RAILS_ENV} \
    RUBY_GC_HEAP_GROWTH_MAX_SLOTS=${RUBY_GC_HEAP_GROWTH_MAX_SLOTS} \
    RUBY_GC_HEAP_INIT_SLOTS=${RUBY_GC_HEAP_INIT_SLOTS} \
    RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=${RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR} \
    UNICORN_SIDEKIQS=${UNICORN_SIDEKIQS} \
    UNICORN_WORKERS=${UNICORN_WORKERS}
EXPOSE 443
EXPOSE 80
EXPOSE 90
COPY config.yaml /temp-config.yaml
RUN cat /temp-config.yaml | /usr/local/bin/pups --skip-tags=precompile,migrate,db --stdin && rm /temp-config.yaml
CMD ["/sbin/boot"]`))

		Expect(dockerfile).ToNot(ContainSubstring(`discourse-builder`))
		Expect(dockerfile).ToNot(ContainSubstring(`discourse-slim`))
	})

	It("can generate a dockerfile with all env baked into the image", func() {
		dockerfile := conf.Dockerfile(true, false, "config.yaml")
		Expect(dockerfile).To(ContainSubstring(`FROM ${dockerfile_from_image} AS discourse-full
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
ENV LANG=${LANG} \
    LANGUAGE=${LANGUAGE} \
    LC_ALL=${LC_ALL} \
    MULTI=${MULTI} \
    RAILS_ENV=${RAILS_ENV} \
    REPLACED=${REPLACED} \
    RUBY_GC_HEAP_GROWTH_MAX_SLOTS=${RUBY_GC_HEAP_GROWTH_MAX_SLOTS} \
    RUBY_GC_HEAP_INIT_SLOTS=${RUBY_GC_HEAP_INIT_SLOTS} \
    RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=${RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR} \
    UNICORN_SIDEKIQS=${UNICORN_SIDEKIQS} \
    UNICORN_WORKERS=${UNICORN_WORKERS}
EXPOSE 443
EXPOSE 80
EXPOSE 90
COPY config.yaml /temp-config.yaml
RUN cat /temp-config.yaml | /usr/local/bin/pups --skip-tags=precompile,migrate,db --stdin && rm /temp-config.yaml
CMD ["/sbin/boot"]`))
		Expect(dockerfile).ToNot(ContainSubstring(`discourse-builder`))
		Expect(dockerfile).ToNot(ContainSubstring(`discourse-slim`))
	})

	It("can generate configuration for a slim image from a multistage build", func() {
		dockerfile := conf.Dockerfile(false, true, "config.yaml")
		Expect(dockerfile).To(ContainSubstring(`FROM ${dockerfile_from_image} AS discourse-full
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
ENV RAILS_ENV=${RAILS_ENV} \
    RUBY_GC_HEAP_GROWTH_MAX_SLOTS=${RUBY_GC_HEAP_GROWTH_MAX_SLOTS} \
    RUBY_GC_HEAP_INIT_SLOTS=${RUBY_GC_HEAP_INIT_SLOTS} \
    RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=${RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR} \
    UNICORN_SIDEKIQS=${UNICORN_SIDEKIQS} \
    UNICORN_WORKERS=${UNICORN_WORKERS}
EXPOSE 443
EXPOSE 80
EXPOSE 90
COPY config.yaml /temp-config.yaml
RUN cat /temp-config.yaml | /usr/local/bin/pups --skip-tags=precompile,migrate,db --stdin && rm /temp-config.yaml
CMD ["/sbin/boot"]

FROM discourse-full AS discourse-builder
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
RUN GIT_HASH=$(sudo -u discourse git -C /var/www/discourse rev-parse HEAD) &&\
FULL_VERSION=$(sudo -u discourse git -C /var/www/discourse describe --dirty --match "v[0-9]*" 2> /dev/null) &&\
GIT_BRANCH=$(sudo -u discourse git -C /var/www/discourse branch --show-current) &&\
printf '{"git_version":"%s", "full_version":"%s","git_branch":"%s"}' "${GIT_HASH}" "${FULL_VERSION}" "${GIT_BRANCH}" > /var/www/discourse/config/git-utils-overrides.json

FROM ${dockerfile_from_image_slim} AS discourse-slim
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
ENV RAILS_ENV=${RAILS_ENV} \
    RUBY_GC_HEAP_GROWTH_MAX_SLOTS=${RUBY_GC_HEAP_GROWTH_MAX_SLOTS} \
    RUBY_GC_HEAP_INIT_SLOTS=${RUBY_GC_HEAP_INIT_SLOTS} \
    RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=${RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR} \
    UNICORN_SIDEKIQS=${UNICORN_SIDEKIQS} \
    UNICORN_WORKERS=${UNICORN_WORKERS}
EXPOSE 443
EXPOSE 80
EXPOSE 90
COPY config.yaml /temp-config.yaml
COPY --chown=discourse:discourse --from=discourse-builder --exclude=.git --exclude=tmp --exclude=**/node_modules --exclude=**/libv8_monolith.a /var/www/discourse/ /var/www/discourse`))
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
})
