package db

// Port
type Port struct {
	Name     string `yaml:"name"`
	Protocol string `yaml:"protocol"`
	Port     int    `yaml:"port"`
}

// Cmd
type Cmd struct {
	Command []string `yaml:"command"`
	Args    []string `yaml:"args"`
}

// DeployTaskConfig
type DeployTaskConfig struct {
	Envs       map[string]interface{} `yaml:"envs"`
	Ports      []Port                 `yaml:"ports"`
	Stateful   bool                   `yaml:"stateful"`
	Storage    bool                   `yaml:"storage"`
	WorkingDir string                 `yaml:"workingDir"`
	Cmd        Cmd                    `yaml:"cmd"`
}
