package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/Wing924/shellwords"
	"github.com/discourse/launcher/v2/config"
	"github.com/discourse/launcher/v2/utils"
)

type DockerBuilder struct {
	Config    *config.Config
	Stdin     io.Reader
	Dir       string
	Namespace string
	ImageTag  string
}

func (r *DockerBuilder) Run(ctx context.Context) error {
	if r.ImageTag == "" {
		r.ImageTag = "latest"
	}
	cmd := exec.CommandContext(ctx, utils.DockerPath, "build")
	TimeoutDockerBuild(cmd)
	cmd.Dir = r.Dir
	cmd.Env = os.Environ()
	env := r.Config.GetEnvSlice(false)
	cmd.Env = append(cmd.Env, env...)
	cmd.Env = append(cmd.Env, "BUILDKIT_PROGRESS=plain")
	for k := range r.Config.Env {
		cmd.Args = append(cmd.Args, "--build-arg")
		cmd.Args = append(cmd.Args, k)
	}
	cmd.Args = append(cmd.Args, "--no-cache")
	cmd.Args = append(cmd.Args, "--pull")
	cmd.Args = append(cmd.Args, "--force-rm")
	cmd.Args = append(cmd.Args, "-t")
	cmd.Args = append(cmd.Args, r.Namespace+"/"+r.Config.Name+":"+r.ImageTag)
	cmd.Args = append(cmd.Args, "--shm-size=512m")
	cmd.Args = append(cmd.Args, "-f")
	cmd.Args = append(cmd.Args, "-")
	cmd.Args = append(cmd.Args, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = r.Stdin
	if err := utils.CmdRunner(cmd).Run(); err != nil {
		return err
	}
	return nil
}

type DockerRunner struct {
	Config      *config.Config
	ExtraEnv    []string
	ExtraFlags  []string
	Rm          bool
	ContainerId string
	CustomImage string
	Cmd         []string
	Stdin       io.Reader
	SkipPorts   bool
	DryRun      bool
	Restart     bool
	Detatch     bool
	Hostname    string
}

func (r *DockerRunner) Run(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, utils.DockerPath, "run")

	// Detatch signifies we do not want to supervise
	if !r.Detatch {
		TimeoutDockerContainer(cmd, r.ContainerId)
	}

	cmd.Env = os.Environ()
	env := r.Config.GetEnvSlice(true)
	cmd.Env = append(cmd.Env, env...)
	envKeys := make([]string, 0, len(r.Config.Env))

	for envKey := range r.Config.Env {
		envKeys = append(envKeys, envKey)
	}

	sort.Strings(envKeys)

	if r.DryRun {
		// multi-line env doesn't work super great from CLI, but we can print out the rest.
		for _, envKey := range envKeys {
			value := r.Config.Env[envKey]

			if !strings.Contains(value, "\n") {
				cmd.Args = append(cmd.Args, "--env")
				cmd.Args = append(cmd.Args, envKey+"="+shellwords.Escape(value))
			}
		}
	} else {
		for _, envKey := range envKeys {
			cmd.Args = append(cmd.Args, "--env")
			cmd.Args = append(cmd.Args, envKey)
		}
	}

	// Order is important here, we add extra env after config's env to override anything set in env.
	for _, e := range r.ExtraEnv {
		cmd.Args = append(cmd.Args, "--env")
		cmd.Args = append(cmd.Args, e)
	}

	for k, v := range r.Config.Labels {
		cmd.Args = append(cmd.Args, "--label")
		cmd.Args = append(cmd.Args, k+"="+v)
	}

	if !r.SkipPorts {
		for _, v := range r.Config.Expose {
			if strings.Contains(v, ":") {
				cmd.Args = append(cmd.Args, "--publish")
				cmd.Args = append(cmd.Args, v)
			} else {
				cmd.Args = append(cmd.Args, "--expose")
				cmd.Args = append(cmd.Args, v)
			}
		}
	}

	for _, v := range r.Config.Volumes {
		cmd.Args = append(cmd.Args, "--volume")
		cmd.Args = append(cmd.Args, v.Volume.Host+":"+v.Volume.Guest)
	}

	for _, v := range r.Config.Links {
		cmd.Args = append(cmd.Args, "--link")
		cmd.Args = append(cmd.Args, v.Link.Name+":"+v.Link.Alias)
	}

	cmd.Args = append(cmd.Args, "--shm-size=512m")

	if r.Rm {
		cmd.Args = append(cmd.Args, "--rm")
	}

	if r.Restart {
		cmd.Args = append(cmd.Args, "--restart=always")
	} else {
		cmd.Args = append(cmd.Args, "--restart=no")
	}

	if r.Detatch {
		cmd.Args = append(cmd.Args, "--detach")
	}

	cmd.Args = append(cmd.Args, "--interactive")

	// Docker args override settings above
	cmd.Args = append(cmd.Args, r.Config.GetDockerArgs()...)
	cmd.Args = append(cmd.Args, r.ExtraFlags...)

	if r.Hostname != "" {
		cmd.Args = append(cmd.Args, "--hostname")
		cmd.Args = append(cmd.Args, r.Hostname)
	}

	cmd.Args = append(cmd.Args, "--name")
	cmd.Args = append(cmd.Args, r.ContainerId)

	if len(r.CustomImage) > 0 {
		cmd.Args = append(cmd.Args, r.CustomImage)
	} else {
		cmd.Args = append(cmd.Args, r.Config.GetRunImage())
	}

	cmd.Args = append(cmd.Args, r.Cmd...)

	if !r.Detatch {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = r.Stdin
	}

	runner := utils.CmdRunner(cmd)

	if r.DryRun {
		fmt.Println(cmd)
	} else {
		if err := runner.Run(); err != nil {
			return err
		}
	}
	return nil
}

type DockerPupsRunner struct {
	Config         *config.Config
	PupsArgs       string
	FromImageName  string
	SavedImageName string
	ExtraEnv       []string
	ContainerId    string
}

func (r *DockerPupsRunner) Run(ctx context.Context) error {
	rm := false
	// remove : in case docker tag is blank, and use default latest tag
	r.SavedImageName = strings.TrimRight(r.SavedImageName, ":")

	if r.SavedImageName == "" {
		rm = true
	}

	defer func(rm bool) {
		if !rm {
			time.Sleep(utils.CommitWait)
			runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			cmd := exec.CommandContext(runCtx, utils.DockerPath, "rm", "--force", r.ContainerId)
			if err := utils.CmdRunner(cmd).Run(); err != nil {
				fmt.Fprintln(utils.Out, "Error stopping container"+r.ContainerId) //nolint:errcheck
			}
			cancel()
		}
	}(rm)

	commands := []string{
		"/bin/bash",
		"-c",
		"/usr/local/bin/pups --stdin " + r.PupsArgs,
	}

	runner := DockerRunner{Config: r.Config,
		ExtraEnv:    r.ExtraEnv,
		Rm:          rm,
		CustomImage: r.FromImageName,
		ContainerId: r.ContainerId,
		Cmd:         commands,
		Stdin:       strings.NewReader(r.Config.Yaml()),
		SkipPorts:   true, //pups runs don't need to expose ports
	}

	if err := runner.Run(ctx); err != nil {
		return err
	}

	if len(r.SavedImageName) > 0 {
		time.Sleep(utils.CommitWait)

		cmd := exec.Command(utils.DockerPath,
			"commit",
			"--change",
			"LABEL org.opencontainers.image.created=\""+time.Now().UTC().Format(time.RFC3339)+"\"",
			"--change",
			"CMD [\""+r.Config.GetBootCommand()+"\"]",
			r.ContainerId,
			r.SavedImageName,
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		fmt.Fprintln(utils.Out, cmd) //nolint:errcheck

		if err := utils.CmdRunner(cmd).Run(); err != nil {
			return err
		}
	}

	return nil
}

func ContainerExists(container string) (bool, error) {
	cmd := exec.Command(utils.DockerPath, "ps", "--all", "--quiet", "--filter", "name="+container)
	result, err := utils.CmdRunner(cmd).Output()

	if err != nil {
		return false, err
	}

	if len(result) > 0 {
		return true, nil
	}

	return false, nil
}

func ContainerRunning(container string) (bool, error) {
	cmd := exec.Command(utils.DockerPath, "ps", "--quiet", "--filter", "name="+container)
	result, err := utils.CmdRunner(cmd).Output()

	if err != nil {
		return false, err
	}

	if len(result) > 0 {
		return true, nil
	}

	return false, nil
}
