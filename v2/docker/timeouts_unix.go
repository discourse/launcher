//go:build !windows

package docker

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/discourse/launcher/v2/utils"
)

func TimeoutDockerBuild(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return unix.Kill(-cmd.Process.Pid, unix.SIGINT)
	}
}

func TimeoutDockerContainer(cmd *exec.Cmd, containerId string) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// MacOS cannot kill a process group using the negative pid.
		// attempt to stop a container by running docker stop
		if runtime.GOOS == "darwin" {
			runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			stopCmd := exec.CommandContext(runCtx, utils.DockerPath, "stop", containerId)
			if err := utils.CmdRunner(stopCmd).Run(); err != nil {
				fmt.Fprintln(utils.Out, "Error stopping container"+containerId) //nolint:errcheck
			}
			cancel()
		}
		return unix.Kill(-cmd.Process.Pid, unix.SIGINT)
	}
}
