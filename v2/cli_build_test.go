package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	ddocker "github.com/discourse/launcher/v2"
	. "github.com/discourse/launcher/v2/test_utils"
	"github.com/discourse/launcher/v2/utils"
)

var _ = Describe("Build", func() {
	var testDir string
	var out *bytes.Buffer
	var cli *ddocker.Cli
	var ctx context.Context

	BeforeEach(func() {
		utils.DockerPath = "docker"
		out = &bytes.Buffer{}
		utils.Out = out
		testDir, _ = os.MkdirTemp("", "ddocker-test")

		ctx = context.Background()

		cli = &ddocker.Cli{
			ConfDir:      "./test/containers",
			TemplatesDir: "./test",
			BuildDir:     testDir,
		}

		utils.CmdRunner = CreateNewFakeCmdRunner()
	})

	Context("When running build commands", func() {
		var checkBuildCmd = func(cmd exec.Cmd) {
			Expect(cmd.String()).To(ContainSubstring("docker build"))
			//secret build args envs are ignored
			Expect(cmd.String()).ToNot(ContainSubstring("--build-arg DISCOURSE_DB_PASSWORD"))
			Expect(cmd.String()).To(ContainSubstring("--build-arg RUBY_GC_HEAP_INIT_SLOTS"))
			Expect(cmd.Dir).To(Equal(testDir))

			//db password is ignored
			Expect(cmd.Env).ToNot(ContainElement("DISCOURSE_DB_PASSWORD=SOME_SECRET"))
			Expect(cmd.Env).ToNot(ContainElement("DISCOURSEDB_SOCKET="))
			buf := new(strings.Builder)
			io.Copy(buf, cmd.Stdin) //nolint:errcheck
			// docker build's stdin is a dockerfile
			Expect(buf.String()).To(ContainSubstring("COPY config.yaml /temp-config.yaml"))
			Expect(buf.String()).To(ContainSubstring("--skip-tags=precompile,migrate,db"))
			Expect(buf.String()).ToNot(ContainSubstring("SKIP_EMBER_CLI_COMPILE=1"))

			// Ensure we clean up the temp dir after building
			_, err := os.Stat(testDir)
			Expect(err).To(MatchError(os.IsNotExist, "IsNotExist"))
		}

		var checkMigrateCmd = func(cmd exec.Cmd) {
			Expect(cmd.String()).To(ContainSubstring("docker run"))
			Expect(cmd.String()).To(ContainSubstring("--env DISCOURSE_DEVELOPER_EMAILS"))
			Expect(cmd.String()).To(ContainSubstring("--env SKIP_EMBER_CLI_COMPILE=1"))
			// no commit after, we expect an --rm as the container isn't needed after it is stopped
			Expect(cmd.String()).To(ContainSubstring("--rm"))
			Expect(cmd.Env).To(ContainElement("DISCOURSE_DB_PASSWORD=SOME_SECRET"))
			buf := new(strings.Builder)
			io.Copy(buf, cmd.Stdin) //nolint:errcheck
			// docker run's stdin is a pups config
			Expect(buf.String()).To(ContainSubstring("path: /etc/service/nginx/run"))
		}

		var checkConfigureCmd = func(cmd exec.Cmd) {
			Expect(cmd.String()).To(ContainSubstring(
				"docker run " +
					"--env DISCOURSE_DB_HOST " +
					"--env DISCOURSE_DB_PASSWORD " +
					"--env DISCOURSE_DB_PORT " +
					"--env DISCOURSE_DB_SOCKET " +
					"--env DISCOURSE_DEVELOPER_EMAILS " +
					"--env DISCOURSE_HOSTNAME " +
					"--env DISCOURSE_REDIS_HOST " +
					"--env DISCOURSE_SMTP_ADDRESS " +
					"--env DISCOURSE_SMTP_PASSWORD " +
					"--env DISCOURSE_SMTP_USER_NAME " +
					"--env LANG " +
					"--env LANGUAGE " +
					"--env LC_ALL " +
					"--env MULTI " +
					"--env RAILS_ENV " +
					"--env REPLACED " +
					"--env RUBY_GC_HEAP_GROWTH_MAX_SLOTS " +
					"--env RUBY_GC_HEAP_INIT_SLOTS " +
					"--env RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR " +
					"--env UNICORN_SIDEKIQS " +
					"--env UNICORN_WORKERS " +
					"--env SKIP_EMBER_CLI_COMPILE=1 " +
					"--volume /var/discourse/shared/web-only:/shared " +
					"--volume /var/discourse/shared/web-only/log/var-log:/var/log " +
					"--link data:data " +
					"--shm-size=512m " +
					"--restart=no " +
					"--interactive " +
					"--expose 100 " +
					"--name discourse-build-test " +
					"local_discourse/test /bin/bash -c /usr/local/bin/pups --stdin --tags=db,precompile",
			))

			Expect(cmd.Env).To(ContainElements(
				"DISCOURSE_DB_HOST=data",
				"DISCOURSE_DB_PASSWORD=SOME_SECRET",
				"DISCOURSE_DB_PORT=",
				"DISCOURSE_DB_SOCKET=",
				"DISCOURSE_DEVELOPER_EMAILS=me@example.com,you@example.com",
				"DISCOURSE_HOSTNAME=discourse.example.com",
				"DISCOURSE_REDIS_HOST=data",
				"DISCOURSE_SMTP_ADDRESS=smtp.example.com",
				"DISCOURSE_SMTP_PASSWORD=pa$$word",
				"DISCOURSE_SMTP_USER_NAME=user@example.com",
				"LANG=en_US.UTF-8",
				"LANGUAGE=en_US.UTF-8",
				"LC_ALL=en_US.UTF-8",
				"MULTI=test\nmultiline with some spaces\nvar\n",
				"RAILS_ENV=production",
				"REPLACED=test/test/test",
				"RUBY_GC_HEAP_GROWTH_MAX_SLOTS=40000",
				"RUBY_GC_HEAP_INIT_SLOTS=400000",
				"RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=1.5",
				"UNICORN_SIDEKIQS=1",
				"UNICORN_WORKERS=3",
			))

			buf := new(strings.Builder)
			io.Copy(buf, cmd.Stdin) //nolint:errcheck
			// docker run's stdin is a pups config

			// web.template.yml is merged with the test config
			Expect(buf.String()).To(ContainSubstring("path: /etc/service/nginx/run"))
			Expect(buf.String()).To(ContainSubstring(`exec: echo "custom test command"`))
		}

		// commit on configure
		var checkConfigureCommit = func(cmd exec.Cmd) {
			Expect(cmd.String()).To(MatchRegexp(
				"docker commit " +
					`--change LABEL org\.opencontainers\.image\.created="[\d\-T:Z]+" ` +
					`--change CMD \["/sbin/boot"\] ` +
					"discourse-build-test local_discourse/test",
			))

			Expect(cmd.Env).To(BeNil())
		}

		// configure also cleans up
		var checkConfigureClean = func(cmd exec.Cmd) {
			Expect(cmd.String()).To(ContainSubstring("docker rm --force discourse-build-test"))
		}

		It("Should run docker build with correct arguments", func() {
			runner := ddocker.DockerBuildCmd{Config: "test"}
			runner.Run(cli, ctx) //nolint:errcheck
			Expect(len(RanCmds)).To(Equal(1))
			checkBuildCmd(RanCmds[0])
		})

		It("Should allow for extra build args", func() {
			runner := ddocker.DockerBuildCmd{Config: "test", ExtraFlags: []string{"--platform", "linux/amd64,linux/arm64"}}
			runner.Run(cli, ctx) //nolint:errcheck
			Expect(len(RanCmds)).To(Equal(1))
			checkBuildCmd(RanCmds[0])
			Expect(RanCmds[0].String()).To(ContainSubstring("--platform linux/amd64,linux/arm64"))
		})

		It("Should run docker migrate with correct arguments", func() {
			runner := ddocker.DockerMigrateCmd{Config: "test"}
			runner.Run(cli, ctx) //nolint:errcheck
			Expect(len(RanCmds)).To(Equal(1))
			checkMigrateCmd(RanCmds[0])
		})

		Context("With a custom namespace", func() {
			BeforeEach(func() {
				cli.Namespace = "testnamespace"
			})

			It("Should run docker build with correct namespace and custom flags", func() {
				runner := ddocker.DockerBuildCmd{Config: "test"}
				runner.Run(cli, ctx) //nolint:errcheck
				Expect(len(RanCmds)).To(Equal(1))
				checkBuildCmd(RanCmds[0])
				Expect(RanCmds[0].String()).To(ContainSubstring("testnamespace/test"))
			})

			It("Should run docker build with custom tag", func() {
				runner := ddocker.DockerBuildCmd{Config: "test", Tag: "custom/tag"}
				runner.Run(cli, ctx) //nolint:errcheck
				Expect(len(RanCmds)).To(Equal(1))
				checkBuildCmd(RanCmds[0])
				Expect(RanCmds[0].String()).To(ContainSubstring("custom/tag"))
			})

			It("Should build with both custom tags set", func() {
				runner := ddocker.DockerBuildCmd{Config: "test", Tag: "custom/tag", ExtraFlags: []string{"-t", "custom_docker/tag2"}}
				runner.Run(cli, ctx) //nolint:errcheck
				Expect(len(RanCmds)).To(Equal(1))
				checkBuildCmd(RanCmds[0])
				Expect(RanCmds[0].String()).To(ContainSubstring("custom/tag"))
				Expect(RanCmds[0].String()).To(ContainSubstring("custom_docker/tag2"))
			})

			var CheckBuildWithDockerBuildTagOnly = func(dockerBuildFlags []string) {
				runner := ddocker.DockerBuildCmd{Config: "test", ExtraFlags: dockerBuildFlags}
				runner.Run(cli, ctx) //nolint:errcheck
				Expect(len(RanCmds)).To(Equal(1))
				checkBuildCmd(RanCmds[0])
				Expect(RanCmds[0].String()).ToNot(ContainSubstring("testnamespace/test"))
				Expect(RanCmds[0].String()).To(ContainSubstring("custom/tag"))
			}
			It("Should omit the default tag when only docker build tag is passed as -t=", func() {
				CheckBuildWithDockerBuildTagOnly([]string{"-t=custom/tag"})
			})
			It("Should omit the default tag when only docker build tag is passed as -t", func() {
				CheckBuildWithDockerBuildTagOnly([]string{"-t", "custom/tag"})
			})
			It("Should omit the default tag when only docker build tag is passed as --tag=", func() {
				CheckBuildWithDockerBuildTagOnly([]string{"--tag=custom/tag"})
			})
			It("Should omit the default tag when only docker build tag is passed as --tag", func() {
				CheckBuildWithDockerBuildTagOnly([]string{"--tag", "custom/tag"})
			})

			It("Should run docker configure with correct namespace and tags", func() {
				runner := ddocker.DockerConfigureCmd{Config: "test", SourceTag: "source/build"}
				runner.Run(cli, ctx) //nolint:errcheck
				Expect(len(RanCmds)).To(Equal(3))

				Expect(RanCmds[0].String()).To(MatchRegexp(
					"--name discourse-build-test " +
						"source/build /bin/bash -c /usr/local/bin/pups --stdin --tags=db,precompile",
				))
				Expect(RanCmds[1].String()).To(MatchRegexp(
					"docker commit " +
						`--change LABEL org\.opencontainers\.image\.created="[\d\-T:Z]+" ` +
						`--change CMD \["/sbin/boot"\] ` +
						"discourse-build-test testnamespace/test",
				))
				checkConfigureClean(RanCmds[2])
			})

			It("Should run docker migrate with correct namespace", func() {
				runner := ddocker.DockerMigrateCmd{Config: "test"}
				runner.Run(cli, ctx) //nolint:errcheck
				Expect(len(RanCmds)).To(Equal(1))
				Expect(RanCmds[0].String()).To(ContainSubstring("testnamespace/test "))
			})
		})

		It("Should allow skip post deployment migrations", func() {
			runner := ddocker.DockerMigrateCmd{Config: "test", SkipPostDeploymentMigrations: true}
			runner.Run(cli, ctx) //nolint:errcheck
			Expect(len(RanCmds)).To(Equal(1))
			cmd := RanCmds[0]
			Expect(cmd.String()).To(ContainSubstring("docker run"))
			Expect(cmd.String()).To(ContainSubstring("--env DISCOURSE_DEVELOPER_EMAILS"))
			Expect(cmd.String()).To(ContainSubstring("--env SKIP_POST_DEPLOYMENT_MIGRATIONS=1"))
			Expect(cmd.String()).To(ContainSubstring("--env SKIP_EMBER_CLI_COMPILE=1"))
			// no commit after, we expect an --rm as the container isn't needed after it is stopped
			Expect(cmd.String()).To(ContainSubstring("--rm"))
			Expect(cmd.Env).To(ContainElement("DISCOURSE_DB_PASSWORD=SOME_SECRET"))
			buf := new(strings.Builder)
			io.Copy(buf, cmd.Stdin) //nolint:errcheck
			// docker run's stdin is a pups config
			Expect(buf.String()).To(ContainSubstring("path: /etc/service/nginx/run"))
		})

		It("Should run docker run followed by docker commit and rm container when configuring", func() {
			runner := ddocker.DockerConfigureCmd{Config: "test"}
			runner.Run(cli, ctx) //nolint:errcheck
			Expect(len(RanCmds)).To(Equal(3))

			checkConfigureCmd(RanCmds[0])
			checkConfigureCommit(RanCmds[1])
			checkConfigureClean(RanCmds[2])
		})

		It("Should run all docker commands for full bootstrap", func() {
			runner := ddocker.DockerBootstrapCmd{Config: "test"}
			runner.Run(cli, ctx) //nolint:errcheck
			Expect(len(RanCmds)).To(Equal(5))
			checkBuildCmd(RanCmds[0])
			checkMigrateCmd(RanCmds[1])
			checkConfigureCmd(RanCmds[2])
			checkConfigureCommit(RanCmds[3])
			checkConfigureClean(RanCmds[4])
		})
	})
})
