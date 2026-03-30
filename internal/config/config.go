package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Project holds credentials for a single Jira instance.
type Project struct {
	Email  string `toml:"email"`
	Token  string `toml:"token"`
	Server string `toml:"server"`
}

// Config holds multi-project configuration.
type Config struct {
	DefaultProject string              `toml:"default_project,omitempty"`
	Projects       map[string]*Project `toml:"projects,omitempty"`
}

// Credentials holds resolved auth credentials.
type Credentials struct {
	Email  string
	Token  string
	Server string
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "jx", "config.toml"), nil
}

func loadConfigFile() (*Config, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfigFile(cfg *Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func resolveProject(cfg *Config, projectFlag string) *Project {
	if cfg == nil {
		return nil
	}
	if projectFlag != "" && cfg.Projects != nil {
		if p, ok := cfg.Projects[projectFlag]; ok {
			return p
		}
		return nil
	}
	if cfg.DefaultProject != "" && cfg.Projects != nil {
		if p, ok := cfg.Projects[cfg.DefaultProject]; ok {
			return p
		}
	}
	return nil
}

// LoadCredentials resolves credentials from flag > env > config file.
func LoadCredentials(emailFlag, tokenFlag, serverFlag, projectFlag string) (*Credentials, error) {
	creds := &Credentials{}

	// Try config file first for defaults
	cfg, _ := loadConfigFile()
	if p := resolveProject(cfg, projectFlag); p != nil {
		creds.Email = p.Email
		creds.Token = p.Token
		creds.Server = p.Server
	}

	// Env vars override config
	if v := os.Getenv("JIRA_EMAIL"); v != "" {
		creds.Email = v
	}
	if v := os.Getenv("JIRA_API_TOKEN"); v != "" {
		creds.Token = v
	}
	if v := os.Getenv("JIRA_SERVER"); v != "" {
		creds.Server = v
	}

	// Flags override everything
	if emailFlag != "" {
		creds.Email = emailFlag
	}
	if tokenFlag != "" {
		creds.Token = tokenFlag
	}
	if serverFlag != "" {
		creds.Server = serverFlag
	}

	if creds.Email == "" {
		return nil, fmt.Errorf("email required: use --email flag, JIRA_EMAIL env var, or run 'jx config add'")
	}
	if creds.Token == "" {
		return nil, fmt.Errorf("API token required: use --token flag, JIRA_API_TOKEN env var, or run 'jx config add'")
	}
	if creds.Server == "" {
		return nil, fmt.Errorf("server URL required: use --server flag, JIRA_SERVER env var, or run 'jx config add'")
	}

	return creds, nil
}

// AddProject saves a named project to the config file.
func AddProject(name, email, token, server string) error {
	cfg, err := loadConfigFile()
	if err != nil {
		cfg = &Config{}
	}
	if cfg.Projects == nil {
		cfg.Projects = make(map[string]*Project)
	}
	cfg.Projects[name] = &Project{
		Email:  email,
		Token:  token,
		Server: server,
	}
	if cfg.DefaultProject == "" {
		cfg.DefaultProject = name
	}
	return saveConfigFile(cfg)
}

// RemoveProject deletes a named project from the config file.
func RemoveProject(name string) error {
	cfg, err := loadConfigFile()
	if err != nil {
		return fmt.Errorf("no config file found")
	}
	if cfg.Projects == nil {
		return fmt.Errorf("project %q not found", name)
	}
	if _, ok := cfg.Projects[name]; !ok {
		return fmt.Errorf("project %q not found", name)
	}
	delete(cfg.Projects, name)
	if cfg.DefaultProject == name {
		cfg.DefaultProject = ""
		for k := range cfg.Projects {
			cfg.DefaultProject = k
			break
		}
	}
	if len(cfg.Projects) == 0 {
		cfg.Projects = nil
	}
	return saveConfigFile(cfg)
}

// SetDefaultProject sets the default project.
func SetDefaultProject(name string) error {
	cfg, err := loadConfigFile()
	if err != nil {
		return fmt.Errorf("no config file found")
	}
	if cfg.Projects == nil {
		return fmt.Errorf("project %q not found", name)
	}
	if _, ok := cfg.Projects[name]; !ok {
		return fmt.Errorf("project %q not found", name)
	}
	cfg.DefaultProject = name
	return saveConfigFile(cfg)
}

// ListProjects returns the loaded config.
func ListProjects() (*Config, error) {
	return loadConfigFile()
}

// MaskKey masks sensitive tokens for display.
func MaskKey(key string) string {
	if len(key) <= 10 {
		return "***"
	}
	return key[:8] + "***" + key[len(key)-4:]
}
