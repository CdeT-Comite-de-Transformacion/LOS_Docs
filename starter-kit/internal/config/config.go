package config

const DefaultFilename = "modkit.yaml"

type Config struct {
	Project  ProjectConfig  `yaml:"project"`
	Modules  ModulesConfig  `yaml:"modules"`
	Defaults DefaultsConfig `yaml:"defaults"`
}

type ProjectConfig struct {
	Name         string `yaml:"name"`
	Module       string `yaml:"module"`
	Architecture string `yaml:"architecture"`
	Template     string `yaml:"template"`
}

type ModulesConfig struct {
	Root string `yaml:"root"`
}

type DefaultsConfig struct {
	Template string `yaml:"template"`
}

func Default() Config {
	return Config{
		Project: ProjectConfig{
			Name:         "los-cp",
			Module:       "github.com/your-org/los-cp",
			Architecture: "modular-ddd",
			Template:     "http",
		},
		Modules: ModulesConfig{Root: "internal"},
		Defaults: DefaultsConfig{
			Template: "console",
		},
	}
}
