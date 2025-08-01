package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"
	"context"
	"os"

	ddocker "github.com/discourse/launcher/v2"
	. "github.com/discourse/launcher/v2/test_utils"
	"github.com/discourse/launcher/v2/utils"
)

var _ = Describe("Runtime", func() {
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

	Context("When running run commands", func() {
		var checkStartCmd = func() {
			Expect(len(RanCmds)).To(Equal(3))

			cmd := GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker ps --quiet --filter name=test"))

			cmd = GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker ps --all --quiet --filter name=test"))

			cmd = GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker run"))
			Expect(cmd.String()).To(ContainSubstring("--detach"))
			Expect(cmd.String()).To(ContainSubstring("--restart=always"))
			Expect(cmd.String()).To(ContainSubstring("--name test local_discourse/test /sbin/boot"))
		}

		var checkStartCmdWhenStarted = func() {
			Expect(len(RanCmds)).To(Equal(1))

			cmd := GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker ps --quiet --filter name=test"))
		}

		var checkStopCmd = func() {
			Expect(len(RanCmds)).To(Equal(2))

			cmd := GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker ps --all --quiet --filter name=test"))
			cmd = GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker stop --time 600 test"))
		}

		var checkStopCmdWhenMissing = func() {
			Expect(len(RanCmds)).To(Equal(1))

			cmd := GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker ps --all --quiet --filter name=test"))
		}

		Context("without a running container", func() {
			It("should run start commands", func() {
				runner := ddocker.StartCmd{Config: "test"}
				runner.Run(cli, ctx) //nolint:errcheck
				checkStartCmd()
			})

			It("should not run stop commands", func() {
				runner := ddocker.StopCmd{Config: "test"}
				runner.Run(cli, ctx) //nolint:errcheck
				checkStopCmdWhenMissing()
			})
		})

		Context("with a running container", func() {
			BeforeEach(func() {
				//response should be non-empty, indicating a running container
				response := []byte{123}
				CmdOutputResponse = response
			})

			It("should not run start commands", func() {
				runner := ddocker.StartCmd{Config: "test"}
				runner.Run(cli, ctx) //nolint:errcheck
				checkStartCmdWhenStarted()
			})

			It("should run stop commands", func() {
				runner := ddocker.StopCmd{Config: "test"}
				runner.Run(cli, ctx) //nolint:errcheck
				checkStopCmd()
			})

			It("should keep running during commits, and be post-deploy migration aware when using a web only container", func() {
				runner := ddocker.RebuildCmd{Config: "web_only"}
				runner.Run(cli, ctx) //nolint:errcheck

				//initial build
				cmd := GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker build"))

				//migrate, skipping post deployment migrations
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker run"))
				Expect(cmd.String()).To(ContainSubstring("--tags=db,migrate"))
				Expect(cmd.String()).To(ContainSubstring("--env SKIP_POST_DEPLOYMENT_MIGRATIONS=1"))

				// precompile
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker run"))
				Expect(cmd.String()).To(ContainSubstring("--tags=db,precompile"))
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker commit"))
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker rm"))

				// destroying
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker ps --all --quiet --filter name=web_only"))
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker stop --time 600 web_only"))
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker rm"))

				// starting container --run command won't run because
				// tests already believe we're running
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker ps --quiet"))

				// run post-deploy migrations
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker run"))
				Expect(cmd.String()).To(ContainSubstring("--tags=db,migrate"))
				Expect(len(RanCmds)).To(Equal(0))
			})

			It("should stop with standalone", func() {
				runner := ddocker.RebuildCmd{Config: "standalone"}

				runner.Run(cli, ctx) //nolint:errcheck

				//initial build
				cmd := GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker build"))
				cmd = GetLastCommand()

				// stop
				Expect(cmd.String()).To(ContainSubstring("docker ps --all --quiet --filter name=standalone"))
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker stop"))

				// run migrate
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker run"))
				Expect(cmd.String()).To(ContainSubstring("--tags=db,migrate"))
				Expect(cmd.String()).ToNot(ContainSubstring("--env SKIP_POST_DEPLOYMENT_MIGRATIONS=1"))

				// run configure
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker run"))
				Expect(cmd.String()).To(ContainSubstring("--tags=db,precompile"))
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker commit"))
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker rm"))

				// run destroy
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker ps --all --quiet"))
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker stop"))
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker rm standalone"))

				// run start (we think we're already started here so this is just ps)
				cmd = GetLastCommand()
				Expect(cmd.String()).To(ContainSubstring("docker ps --quiet"))
				Expect(len(RanCmds)).To(Equal(0))

				// Ensure we clean up the temp dir after building
				_, err := os.Stat(testDir)
				Expect(err).To(MatchError(os.IsNotExist, "IsNotExist"))
			})
		})

	})
})
