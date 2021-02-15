package main

import (
	"context"
	"os"
	"sync"
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

	files     map[int64]goputio.File
	filesMu   sync.Mutex
	filesGC   map[int64]goputio.File
	filesGCMu sync.Mutex

	downloadCh chan goputio.File
	unzipCh    chan string
}

func (a *app) runDownloadWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case file := <-a.downloadCh:
			a.fetchFile(ctx, file)
		}
	}
}

func (a *app) runUnzipWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case file := <-a.unzipCh:
			a.unzipFile(ctx, file)
		}
	}
}

func (a *app) fetchFile(ctx context.Context, file goputio.File) {
	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	zipfile, err := a.fetchRemoteFile(ctx, file)
	a.filesGCMu.Lock()
	a.filesGC[file.ID] = file
	a.filesGCMu.Unlock()
	if err != nil {
		a.logger.Error("downloading file failed", "error", err)
		return
	}

	a.unzipCh <- zipfile
}

func (a *app) unzipFile(ctx context.Context, zipfile string) {
	err := a.unzipZipfile(zipfile)
	if err != nil {
		a.logger.Error("unzipping file failed", "file", zipfile, "error", err)
	}
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
		a.filesMu.Lock()
		if _, ok := a.files[element.ID]; ok {
			a.filesMu.Unlock()
			continue
		}

		a.logger.Info("adding new file to queue", "file", element.Name, "ID", element.ID)
		a.files[element.ID] = element
		a.filesMu.Unlock()

		a.downloadCh <- element
	}

	for element := range a.filesGC {
		a.filesMu.Lock()
		delete(a.files, element)
		a.filesMu.Unlock()

		a.filesGCMu.Lock()
		delete(a.filesGC, element)
		a.filesGCMu.Unlock()
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
		conf:       *cfg,
		logger:     logger,
		client:     putio,
		files:      map[int64]goputio.File{},
		filesMu:    sync.Mutex{},
		filesGC:    map[int64]goputio.File{},
		filesGCMu:  sync.Mutex{},
		downloadCh: make(chan goputio.File, 10),
		unzipCh:    make(chan string, 2),
	}

	for n := 0; n < 2; n++ {
		go a.runDownloadWorker(context.Background())
	}

	for n := 0; n < 1; n++ {
		go a.runUnzipWorker(context.Background())
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
