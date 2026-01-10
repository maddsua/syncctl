package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	DataDir    string `yaml:"data_dir"`
	HttpPort   int    `yaml:"http_port"`
	TlsPort    int    `yaml:"tls_port"`
	AuthConfig `yaml:",inline"`
}

type AuthConfig struct {
	Users []UserConfig `yaml:"users"`
}

type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	RootDir  string `yaml:"root_dir"`
}

func ReadConfig(configPath string) (*ServerConfig, error) {

	if stat, _ := os.Stat(configPath); stat == nil || !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("config file doesn't exist")
	}

	var cfg ServerConfig

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
