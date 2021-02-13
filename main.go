package main

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/maxeaubrey/putio-getter/putio"
	goputio "github.com/putdotio/go-putio"
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
	client *putio.Putio
}

func (a *app) fetchAndUnzipFile(ctx context.Context, file goputio.File) error {
	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	zipfile, err := a.fetchRemoteFile(ctx, file)
	if err != nil {
		return err
	}

	return a.unzipZipfile(zipfile)
}

func (a *app) downloadAll(ctx context.Context) {
	err := a.client.CleanTransfers(ctx)
	if err != nil {
		a.logger.Error("failed to clean transfers", "error", err)
		return
	}

	list, err := a.client.FetchList(ctx)
	if err != nil {
		a.logger.Error("can't list files in root directory", "error", err)
		return
	}

	for _, element := range list {
		err := a.fetchAndUnzipFile(ctx, element)
		if err != nil {
			return
		}
	}
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "putio-getter",
		Level: hclog.LevelFromString("DEBUG"),
	})

	cfg, err := readConfig()
	if err != nil {
		logger.Error("failed to read configuration", "error", err)
		os.Exit(1)
	}
	err = validateConfig(cfg)
	if err != nil {
		logger.Error("failed to validate configuration", "error", err)
		os.Exit(1)
	}

	logger.SetLevel(hclog.LevelFromString(cfg.LogLevel))

	putio := putio.New(cfg.OauthToken)
	interval, _ := time.ParseDuration(cfg.Interval)

	a := app{
		logger: logger,
		conf:   *cfg,
		client: putio,
	}

	ticker := time.NewTimer(0)
	for {
		select {
		case <-ticker.C:
			a.downloadAll(context.Background())
			ticker.Reset(interval)
		}
	}
}
