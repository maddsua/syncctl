package config

import (
	"encoding/json"
	"os"
	"path"
)

func GetConfigLocation() (string, error) {

	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return path.Join(homedir, ".config/syncctl/state.json"), nil
}

type Config struct {
	Remote   RemoteConfig `json:"remote"`
	Valid    bool         `json:"-"`
	Changed  bool         `json:"-"`
	Location string       `json:"-"`
}

func (config *Config) Store() error {

	loc, err := GetConfigLocation()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path.Dir(loc), os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(loc)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")

	return enc.Encode(*config)
}

func (config *Config) Load() error {

	loc, err := GetConfigLocation()
	if err != nil {
		return err
	}

	if _, err := os.Stat(loc); err != nil {
		return nil
	}

	file, err := os.Open(loc)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(config); err != nil {
		return err
	}

	config.Valid = true
	config.Location = loc

	return nil
}
