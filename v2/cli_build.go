package main

import (
	"context"
	"flag"
	"os"
	"strings"

	"github.com/discourse/launcher/v2/config"
	"github.com/discourse/launcher/v2/docker"
	"github.com/discourse/launcher/v2/utils"
	"github.com/google/uuid"
)

/*
 * build
 * migrate
 * configure
 * bootstrap
 */
type DockerBuildCmd struct {
	Tag        string   `short:"t" help:"Resulting image tag. Defaults to 'local_discourse/{config}'"`
	Config     string   `arg:"" name:"config" help:"configuration" predictor:"config" passthrough:""`
	ExtraFlags []string `arg:"" optional:"" name:"docker-build-flags" help:"Extra build flags for docker build"`
}

func (r *DockerBuildCmd) Run(cli *Cli, ctx context.Context) error {
	config, err := config.LoadConfig(cli.ConfDir, r.Config, true, cli.TemplatesDir)
	if err != nil {
		return err
	}

	dir := cli.BuildDir
	if dir == "" {
		if dir, err = os.MkdirTemp("", "launcher"); err != nil {
			return err
		}
	}
	defer os.RemoveAll(dir) //nolint:errcheck
	configFile := "config.yaml"
	if err := config.WriteYamlConfig(dir, configFile); err != nil {
		return err
	}

	pupsArgs := "--skip-tags=precompile,migrate,db"
	builder := docker.DockerBuilder{
		Config:     config,
		Stdin:      strings.NewReader(config.Dockerfile(pupsArgs, configFile)),
		Dir:        dir,
		ImageTag:   r.Tag,
		ExtraFlags: r.ExtraFlags,
	}
	if err := builder.Run(ctx); err != nil {
		if configErr := config.ValidateConfig(err); configErr != nil {
			return configErr
		}
		return err
	}
	return nil
}

type DockerConfigureCmd struct {
	SourceTag string `short:"s" help:"Source image tag to build from. Defaults to 'local_discourse/{config}'"`
	TargetTag string `short:"t" name:"tag" help:"Target image tag to save as. Defaults to 'local_discourse/{config}'"`
	Config    string `arg:"" name:"config" help:"config" predictor:"config"`
}

func (r *DockerConfigureCmd) Run(cli *Cli, ctx context.Context) error {
	config, err := config.LoadConfig(cli.ConfDir, r.Config, true, cli.TemplatesDir)

	if err != nil {
		return err
	}

	var uuidString string

	if flag.Lookup("test.v") == nil {
		uuidString = uuid.NewString()
	} else {
		uuidString = "test"
	}

	containerId := "discourse-build-" + uuidString
	sourceTag := utils.DefaultNamespace + "/" + r.Config
	if len(r.SourceTag) > 0 {
		sourceTag = r.SourceTag
	}
	targetTag := utils.DefaultNamespace + "/" + r.Config
	if len(r.TargetTag) > 0 {
		targetTag = r.TargetTag
	}

	pups := docker.DockerPupsRunner{
		Config:         config,
		PupsArgs:       "--tags=db,precompile",
		FromImageName:  sourceTag,
		SavedImageName: targetTag,
		ExtraEnv:       []string{"SKIP_EMBER_CLI_COMPILE=1"},
		ContainerId:    containerId,
	}

	return pups.Run(ctx)
}

type DockerMigrateCmd struct {
	Config                       string `arg:"" name:"config" help:"config" predictor:"config"`
	Tag                          string `help:"Image to migrate. Defaults to 'local_discourse/{config}'"`
	SkipPostDeploymentMigrations bool   `env:"SKIP_POST_DEPLOYMENT_MIGRATIONS" help:"Skip post-deployment migrations. Runs safe migrations only. Defers breaking-change migrations. Make sure you run post-deployment migrations after a full deploy is complete if you use this option."`
}

func (r *DockerMigrateCmd) Run(cli *Cli, ctx context.Context) error {
	config, err := config.LoadConfig(cli.ConfDir, r.Config, true, cli.TemplatesDir)
	if err != nil {
		return err
	}
	containerId := "discourse-build-" + uuid.NewString()
	env := []string{"SKIP_EMBER_CLI_COMPILE=1"}
	if r.SkipPostDeploymentMigrations {
		env = append(env, "SKIP_POST_DEPLOYMENT_MIGRATIONS=1")
	}

	tag := utils.DefaultNamespace + "/" + r.Config
	if len(r.Tag) > 0 {
		tag = r.Tag
	}
	pups := docker.DockerPupsRunner{
		Config:        config,
		PupsArgs:      "--tags=db,migrate",
		FromImageName: tag,
		ExtraEnv:      env,
		ContainerId:   containerId,
	}
	return pups.Run(ctx)
}

type DockerBootstrapCmd struct {
	Config string `arg:"" name:"config" help:"config" predictor:"config"`
	Tag    string `short:"t" help:"Resulting image tag. Defaults to 'local_discourse/{config}'"`
}

func (r *DockerBootstrapCmd) Run(cli *Cli, ctx context.Context) error {
	tag := utils.DefaultNamespace + "/" + r.Config
	if len(r.Tag) > 0 {
		tag = r.Tag
	}
	buildStep := DockerBuildCmd{Config: r.Config, Tag: tag}
	migrateStep := DockerMigrateCmd{Config: r.Config, Tag: tag}
	configureStep := DockerConfigureCmd{Config: r.Config, SourceTag: tag, TargetTag: tag}
	if err := buildStep.Run(cli, ctx); err != nil {
		return err
	}
	if err := migrateStep.Run(cli, ctx); err != nil {
		return err
	}
	if err := configureStep.Run(cli, ctx); err != nil {
		return err
	}
	return nil
}
