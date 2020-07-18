package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/hashicorp/go-hclog"
)

func findConfigPath() (string, error) {
	paths := []string{xdg.ConfigHome}
	paths = append(paths, xdg.ConfigDirs...)
	for _, path := range paths {
		file, _ := os.Stat(filepath.Join(path, "putio"))
		if file != nil {
			return filepath.Join(path, "putio"), nil
		}
	}
	return filepath.Join(paths[0], "putio"), fmt.Errorf("no config path exists")
}

func validateConfig(configuration *config) error {
	if configuration.OauthToken == "PLACEHOLDER" {
		return fmt.Errorf("default placeholder token present")
	}

	f, err := os.Stat(configuration.Downloading)
	if os.IsNotExist(err) {
		err = os.MkdirAll(configuration.Downloading, 0755)
		if err != nil {
			return err
		}
	} else if !f.IsDir() {
		return fmt.Errorf("%v already exists and is not a directory", configuration.Downloading)
	}

	f, err = os.Stat(configuration.Unpacking)
	if os.IsNotExist(err) {
		err = os.MkdirAll(configuration.Unpacking, 0755)
		if err != nil {
			return err
		}
	} else if !f.IsDir() {
		return fmt.Errorf("%v already exists and is not a directory", configuration.Downloading)
	}

	_, err = time.ParseDuration(configuration.Interval)
	if err != nil {
		return err
	}

	if hclog.LevelFromString(configuration.LogLevel) == 0 {
		return fmt.Errorf("%v is not a valid log level", configuration.LogLevel)
	}

	return nil
}

func readConfig() (*config, error) {
	configPath, err := findConfigPath()
	if err != nil {
		err := os.MkdirAll(configPath, 0750)
		if err != nil {
			return nil, err
		}
		defaultConfig := &config{
			OauthToken:  "PLACEHOLDER",
			Downloading: filepath.Join(os.TempDir(), "putio-getter"),
			Unpacking:   xdg.UserDirs.Download,
			Interval:    "1m",
			LogLevel:    "ERROR",
		}
		file, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(filepath.Join(configPath, "config.json"), file, 0640)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("configuration doesn't exist, created sample under %v", filepath.Join(configPath, "config.json"))
	}

	configuration := &config{}
	file, err := ioutil.ReadFile(filepath.Join(configPath, "config.json"))
	err = json.Unmarshal(file, &configuration)
	if err != nil {
		return nil, err
	}

	return configuration, nil
}
