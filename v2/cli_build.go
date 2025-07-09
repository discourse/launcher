package main

import (
	"context"
	"errors"
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
	BakeEnv bool   `short:"e" help:"Bake in the configured environment to image after build."`
	Tag     string `default:"latest" help:"Resulting image tag."`

	Config string `arg:"" name:"config" help:"configuration" predictor:"config"`
}

func (r *DockerBuildCmd) Run(cli *Cli, ctx context.Context) error {
	config, err := config.LoadConfig(cli.ConfDir, r.Config, true, cli.TemplatesDir)
	if err != nil {
		return errors.New("YAML syntax error. Please check your containers/*.yml config files")
	}

	dir := cli.BuildDir
	if dir == "" {
		if dir, err = os.MkdirTemp("", "launcher"); err != nil {
			return errors.New("cannot create temp directory")
		}
	}
	defer os.RemoveAll(dir) //nolint:errcheck
	if err := config.WriteYamlConfig(dir); err != nil {
		return err
	}

	namespace := cli.Namespace
	if namespace == "" {
		namespace = utils.DefaultNamespace
	}
	pupsArgs := "--skip-tags=precompile,migrate,db"
	builder := docker.DockerBuilder{
		Config:    config,
		Stdin:     strings.NewReader(config.Dockerfile(pupsArgs, r.BakeEnv)),
		Dir:       dir,
		Namespace: namespace,
		ImageTag:  r.Tag,
	}
	if err := builder.Run(ctx); err != nil {
		return err
	}
	return nil
}

type DockerConfigureCmd struct {
	SourceTag string `help:"Source image tag to build from."`
	TargetTag string `help:"Target image tag to save as."`
	Config    string `arg:"" name:"config" help:"config" predictor:"config"`
}

func (r *DockerConfigureCmd) Run(cli *Cli, ctx context.Context) error {
	config, err := config.LoadConfig(cli.ConfDir, r.Config, true, cli.TemplatesDir)

	if err != nil {
		return errors.New("YAML syntax error. Please check your containers/*.yml config files")
	}

	var uuidString string

	if flag.Lookup("test.v") == nil {
		uuidString = uuid.NewString()
	} else {
		uuidString = "test"
	}

	containerId := "discourse-build-" + uuidString
	namespace := cli.Namespace
	if namespace == "" {
		namespace = utils.DefaultNamespace
	}
	sourceTag := ""
	if len(r.SourceTag) > 0 {
		sourceTag = ":" + r.SourceTag
	}
	targetTag := ""
	if len(r.TargetTag) > 0 {
		targetTag = ":" + r.TargetTag
	}

	pups := docker.DockerPupsRunner{
		Config:         config,
		PupsArgs:       "--tags=db,precompile",
		FromImageName:  namespace + "/" + r.Config + sourceTag,
		SavedImageName: namespace + "/" + r.Config + targetTag,
		ExtraEnv:       []string{"SKIP_EMBER_CLI_COMPILE=1"},
		ContainerId:    containerId,
	}

	return pups.Run(ctx)
}

type DockerMigrateCmd struct {
	Config                       string `arg:"" name:"config" help:"config" predictor:"config"`
	Tag                          string `default:"latest" help:"Image tag to migrate."`
	SkipPostDeploymentMigrations bool   `env:"SKIP_POST_DEPLOYMENT_MIGRATIONS" help:"Skip post-deployment migrations. Runs safe migrations only. Defers breaking-change migrations. Make sure you run post-deployment migrations after a full deploy is complete if you use this option."`
}

func (r *DockerMigrateCmd) Run(cli *Cli, ctx context.Context) error {
	config, err := config.LoadConfig(cli.ConfDir, r.Config, true, cli.TemplatesDir)
	if err != nil {
		return errors.New("YAML syntax error. Please check your containers/*.yml config files")
	}
	containerId := "discourse-build-" + uuid.NewString()
	env := []string{"SKIP_EMBER_CLI_COMPILE=1"}
	if r.SkipPostDeploymentMigrations {
		env = append(env, "SKIP_POST_DEPLOYMENT_MIGRATIONS=1")
	}

	namespace := cli.Namespace
	if namespace == "" {
		namespace = utils.DefaultNamespace
	}
	tag := ""
	if len(r.Tag) > 0 {
		tag = ":" + r.Tag
	}
	pups := docker.DockerPupsRunner{
		Config:        config,
		PupsArgs:      "--tags=db,migrate",
		FromImageName: namespace + "/" + r.Config + tag,
		ExtraEnv:      env,
		ContainerId:   containerId,
	}
	return pups.Run(ctx)
}

type DockerBootstrapCmd struct {
	Config string `arg:"" name:"config" help:"config" predictor:"config"`
}

func (r *DockerBootstrapCmd) Run(cli *Cli, ctx context.Context) error {
	buildStep := DockerBuildCmd{Config: r.Config, BakeEnv: false}
	migrateStep := DockerMigrateCmd{Config: r.Config}
	configureStep := DockerConfigureCmd{Config: r.Config}
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
