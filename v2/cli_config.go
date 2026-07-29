package main

import (
	"context"
	"fmt"
	"os"

	"github.com/discourse/launcher/v2/config"
	"github.com/discourse/launcher/v2/utils"
)

type ResolveCmd struct {
	Config          string            `arg:"" name:"config" help:"config" predictor:"config"`
	Template        string            `short:"T" name:"template" help:"Go text/template rendered against the resolved config. Reference yaml keys, eg '{{.base_image}}' or '{{.env.RAILS_ENV}}'. Omit to print the full resolved YAML."`
	TemplateFile    string            `name:"template-file" help:"Read the template from a file instead of --template." predictor:"file"`
	ConfigOverrides map[string]string `name:"set" help:"Extra config to override values, can override env, params, and base image --set=env.foo=val --set=param.baz=value --set=base_image=override"`
}

func (r *ResolveCmd) Run(cli *Cli, ctx context.Context) error {
	conf, err := config.LoadConfigWithOverrides(cli.ConfDir, r.Config, true, cli.TemplatesDir, r.ConfigOverrides)
	if err != nil {
		return err
	}

	templateText := r.Template
	if r.TemplateFile != "" {
		content, err := os.ReadFile(r.TemplateFile)
		if err != nil {
			return err
		}
		templateText = string(content)
	}

	if templateText == "" {
		out, err := conf.ResolvedYaml()
		if err != nil {
			return err
		}
		fmt.Fprint(utils.Out, out) //nolint:errcheck
		return nil
	}

	out, err := conf.Render(templateText)
	if err != nil {
		return err
	}
	fmt.Fprintln(utils.Out, out) //nolint:errcheck
	return nil
}
