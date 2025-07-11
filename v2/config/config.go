package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/discourse/launcher/v2/utils"

	"gopkg.in/yaml.v3"
)

const defaultBootCommand = "/sbin/boot"

var defaultBakeEnv = []string{
	"RAILS_ENV",
	"UNICORN_WORKERS",
	"UNICORN_SIDEKIQS",
	"RUBY_GC_HEAP_GROWTH_MAX_SLOTS",
	"RUBY_GC_HEAP_INIT_SLOTS",
	"RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR",
	"CREATE_DB_ON_BOOT",
	"MIGRATE_ON_BOOT",
	"PRECOMPILE_ON_BOOT",
}

type Config struct {
	Name          string `yaml:"-"`
	rawYaml       []string
	BaseImage     string            `yaml:"base_image,omitempty"`
	UpdatePups    bool              `yaml:"update_pups,omitempty"`
	RunImage      string            `yaml:"run_image,omitempty"`
	BootCommand   string            `yaml:"boot_command,omitempty"`
	NoBootCommand bool              `yaml:"no_boot_command,omitempty"`
	DockerArgs    string            `yaml:"docker_args,omitempty"`
	Templates     []string          `yaml:"templates,omitempty"`
	Expose        []string          `yaml:"expose,omitempty"`
	Env           map[string]string `yaml:"env,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	Volumes       []struct {
		Volume struct {
			Host  string `yaml:"host"`
			Guest string `yaml:"guest"`
		} `yaml:"volume"`
	} `yaml:"volumes,omitempty"`
	Links []struct {
		Link struct {
			Name  string `yaml:"name"`
			Alias string `yaml:"alias"`
		} `yaml:"link"`
	} `yaml:"links,omitempty"`
}

func (config *Config) loadTemplate(templateDir string, template string) error {
	template_filename := filepath.Join(templateDir, template)
	content, err := os.ReadFile(template_filename)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("template file does not exist: " + template_filename)
		}
		return err
	}
	templateConfig := &Config{}
	if err := yaml.Unmarshal(content, templateConfig); err != nil {
		return err
	}
	if err := mergo.Merge(config, templateConfig, mergo.WithOverride); err != nil {
		return err
	}
	config.rawYaml = append(config.rawYaml, string(content[:]))
	return nil
}

func LoadConfig(dir string, configName string, includeTemplates bool, templatesDir string) (*Config, error) {
	config := &Config{
		Name:        configName,
		BootCommand: defaultBootCommand,
	}

	matched, _ := regexp.MatchString("[[:upper:]/ !@#$%^&*()+~`=]", configName)

	if matched {
		msg := "config name '" + configName + "' must not contain upper case characters, spaces or special characters"
		return nil, errors.New(msg)
	}

	config_filename := filepath.Join(dir, config.Name + ".yml")
	content, err := os.ReadFile(config_filename)

	if err != nil {
		return nil, err
	}

	baseConfig := &Config{}

	if err := yaml.Unmarshal(content, baseConfig); err != nil {
		return nil, err
	}

	if includeTemplates {
		for _, t := range baseConfig.Templates {
			if err := config.loadTemplate(templatesDir, t); err != nil {
				return nil, err
			}
		}
	}

	if err := mergo.Merge(config, baseConfig, mergo.WithOverride); err != nil {
		return nil, err
	}

	config.rawYaml = append(config.rawYaml, string(content[:]))

	if err != nil {
		return nil, err
	}

	for k, v := range config.Labels {
		val := strings.ReplaceAll(v, "{{config}}", config.Name)
		config.Labels[k] = val
	}

	for k, v := range config.Env {
		val := strings.ReplaceAll(v, "{{config}}", config.Name)
		config.Env[k] = val
	}

	if config.BaseImage == "" {
		return nil, errors.New("no base image specified in config, set base image with `base_image: {imagename}`")
	}

	return config, nil
}

func (config *Config) Yaml() string {
	return strings.Join(config.rawYaml, "_FILE_SEPERATOR_")
}

func (config *Config) Dockerfile(pupsArgs string, bakeEnv bool, configFile string) string {
	if configFile == "" {
		configFile = "config.yaml"
	}
	builder := strings.Builder{}
	builder.WriteString("ARG dockerfile_from_image=" + config.BaseImage + "\n")
	builder.WriteString("FROM ${dockerfile_from_image}\n")
	builder.WriteString(config.dockerfileArgs() + "\n")
	if bakeEnv {
		builder.WriteString(config.dockerfileEnvs() + "\n")
	} else {
		builder.WriteString(config.dockerfileDefaultEnvs() + "\n")
	}
	builder.WriteString(config.dockerfileExpose() + "\n")
	builder.WriteString("COPY " + configFile + " /temp-config.yaml\n")
	builder.WriteString("RUN " +
		"cat /temp-config.yaml | /usr/local/bin/pups " + pupsArgs + " --stdin " +
		"&& rm /temp-config.yaml\n")
	builder.WriteString("CMD [\"" + config.GetBootCommand() + "\"]")
	return builder.String()
}

func (config *Config) WriteYamlConfig(dir string, configFile string) error {
	if configFile == "" {
		configFile = "config.yaml"
	}
	file := filepath.Join(dir, configFile)
	if err := os.WriteFile(file, []byte(config.Yaml()), 0660); err != nil {
		return err
	}
	return nil
}

func (config *Config) GetBootCommand() string {
	if len(config.BootCommand) > 0 {
		return config.BootCommand
	} else if config.NoBootCommand {
		return ""
	} else {
		return defaultBootCommand
	}
}

func (config *Config) GetEnvSlice(includeKnownSecrets bool) []string {
	envs := []string{}
	for k, v := range config.Env {
		if !includeKnownSecrets && slices.Contains(utils.KnownSecrets, k) {
			continue
		}
		envs = append(envs, k+"="+v)
	}
	slices.Sort(envs)
	return envs
}

func (config *Config) GetDockerArgs() []string {
	return strings.Fields(config.DockerArgs)
}

func (config *Config) dockerfileEnvs() string {
	builder := []string{}
	for k := range config.Env {
		builder = append(builder, "ENV "+k+"=${"+k+"}")
	}
	slices.Sort(builder)
	return strings.Join(builder, "\n")
}

func (config *Config) dockerfileDefaultEnvs() string {
	builder := []string{}
	for k := range config.Env {
		if slices.Contains(defaultBakeEnv, k) {
			builder = append(builder, "ENV "+k+"=${"+k+"}")
		}
	}
	slices.Sort(builder)
	return strings.Join(builder, "\n")
}

func (config *Config) dockerfileArgs() string {
	builder := []string{}
	for k := range config.Env {
		builder = append(builder, "ARG "+k)
	}
	slices.Sort(builder)
	return strings.Join(builder, "\n")
}

func (config *Config) dockerfileExpose() string {
	builder := []string{}
	for _, p := range config.Expose {
		port := p
		if strings.Contains(p, ":") {
			_, port, _ = strings.Cut(p, ":")
		}
		builder = append(builder, "EXPOSE "+port)
	}
	slices.Sort(builder)
	return strings.Join(builder, "\n")
}

func (config *Config) GetDockerHostname(defaultHostname string) string {
	_, exists := config.Env["DOCKER_USE_HOSTNAME"]
	re := regexp.MustCompile(`[^a-zA-Z-]`)
	hostname := defaultHostname
	if exists {
		hostname = config.Env["DISCOURSE_HOSTNAME"]
	}
	hostname = string(re.ReplaceAll([]byte(hostname), []byte("-"))[:])
	return hostname
}
