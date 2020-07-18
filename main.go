package main

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/putdotio/go-putio"
	"golang.org/x/oauth2"
)

type config struct {
	OauthToken  string
	Downloading string
	Unpacking   string
	Interval    string
	LogLevel    string
}

type app struct {
	conf   config
	logger hclog.Logger
	client *putio.Client
}

// putioRootDir is the directory that we will pull files to download from.
// 0 is a magic value that always corresponds to the current accounts `/` dir,
// but other directories are non-sequential.
const putioRootDir = 0

func (a *app) fetch() {
	list, root, err := a.client.Files.List(context.Background(), putioRootDir)
	if err != nil {
		a.logger.Error("can't list files in root directory", "error", err)
		return
	}

	err = a.client.Transfers.Clean(context.Background())
	if err != nil {
		a.logger.Error("failed to clean transfers", "error", err)
		return
	}

	a.logger.Debug("looking for new files in " + root.Name)
	for _, element := range list {
		err = a.fetchRemoteFile(element)
		if err != nil {
			a.logger.Error("failed to fetch remote file", "error", err)
			return
		}
	}
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "putio-getter",
		Level: hclog.LevelFromString("DEBUG"),
	})

	configuration, err := readConfig()
	if err != nil {
		logger.Error("failed to read configuration", "error", err)
		os.Exit(1)
	}
	err = validateConfig(configuration)
	if err != nil {
		logger.Error("failed to validate configuration", "error", err)
		os.Exit(1)
	}

	logger.SetLevel(hclog.LevelFromString(configuration.LogLevel))

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: configuration.OauthToken})
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	interval, err := time.ParseDuration(configuration.Interval)

	client := putio.NewClient(oauthClient)

	a := app{
		logger: logger,
		conf:   *configuration,
		client: client,
	}

	ticker := time.NewTimer(0)
	for {
		select {
		case <-ticker.C:
			a.fetch()
			ticker.Reset(interval)
		}
	}
}
