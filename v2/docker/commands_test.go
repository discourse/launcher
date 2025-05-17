package docker_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"
	"context"
	"github.com/discourse/launcher/v2/config"
	"github.com/discourse/launcher/v2/docker"
	. "github.com/discourse/launcher/v2/test_utils"
	"github.com/discourse/launcher/v2/utils"
	"os"
	"strings"
)

var _ = Describe("Commands", func() {
	Context("under normal conditions", func() {
		var conf *config.Config
		var out *bytes.Buffer
		var ctx context.Context

		BeforeEach(func() {
			utils.DockerPath = "docker"
			out = &bytes.Buffer{}
			utils.Out = out
			utils.CommitWait = 0
			conf = &config.Config{Name: "test"}
			ctx = context.Background()
			utils.CmdRunner = CreateNewFakeCmdRunner()
		})
		It("Removes unspecified image tags on commit", func() {
			runner := docker.DockerPupsRunner{Config: conf, ContainerId: "123", Ctx: &ctx, SavedImageName: "local_discourse/test:"}
			runner.Run() //nolint:errcheck
			cmd := GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker run"))
			cmd = GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker commit"))
			Expect(strings.HasSuffix(cmd.String(), ":")).To(BeFalse())
			cmd = GetLastCommand()
			Expect(cmd.String()).To(ContainSubstring("docker rm"))
		})

		Context("With environment var set", func() {
			var testDir string
			BeforeEach(func() {
				os.Setenv("launcher_test", "testval") //nolint:errcheck
				testDir, _ = os.MkdirTemp("", "ddocker-test")
			})
			AfterEach(func() {
				os.Unsetenv("launcher_test") //nolint:errcheck
				os.RemoveAll(testDir)        //nolint:errcheck
			})
			It("Inherits environment for docker build", func() {
				runner := docker.DockerBuilder{Config: conf, Ctx: &ctx, Stdin: nil, Dir: testDir, Namespace: "test", ImageTag: "test"}
				runner.Run() //nolint:errcheck
				cmd := GetLastCommand()
				Expect(cmd.Env).To(ContainElement("launcher_test=testval"))
			})
			It("Inherits environment for docker run", func() {
				runner := docker.DockerRunner{Config: conf, Ctx: &ctx, Stdin: nil}
				runner.Run() //nolint:errcheck
				cmd := GetLastCommand()
				Expect(cmd.Env).To(ContainElement("launcher_test=testval"))
			})
		})
	})
})
