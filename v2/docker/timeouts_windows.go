//go:build windows

package docker

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/discourse/launcher/v2/utils"
)

func TimeoutDockerBuild(cmd *exec.Cmd) {
}

func TimeoutDockerContainer(cmd *exec.Cmd, containerId string) {
	cmd.Cancel = func() error {
		// Windows cannot kill a process group using the negative pid.
		// Attempt to stop a container by running docker stop
		runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		stopCmd := exec.CommandContext(runCtx, utils.DockerPath, "stop", containerId)
		if err := utils.CmdRunner(stopCmd).Run(); err != nil {
			fmt.Fprintln(utils.Out, "Error stopping container"+containerId) //nolint:errcheck
		}
		cancel()
		return nil
	}
}
