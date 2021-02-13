package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	goputio "github.com/putdotio/go-putio"
)

func (a *app) fetchRemoteFile(ctx context.Context, file goputio.File) (string, error) {
	a.logger.Info("found " + file.Name)
	zipfile := file.Name + ".zip"

	zipCtx, zipCancelFn := context.WithTimeout(ctx, time.Minute)
	defer zipCancelFn()
	zip, err := a.client.RequestZip(zipCtx, file)
	if err != nil {
		return "", err
	}

	a.logger.Debug("Fetching file", "file", file.Name, "url", zip.URL)
	err = downloadFile(filepath.Join(a.conf.Downloading, zipfile), zip.URL)
	if err != nil {
		return "", err
	}

	a.logger.Debug("Finished downloading. Deleting file.")
	deleteCtx, deleteCancelFn := context.WithTimeout(ctx, time.Minute)
	defer deleteCancelFn()
	err = a.client.DeleteFile(deleteCtx, file.ID)
	if err != nil {
		return "", err
	}

	return zipfile, nil
}

func (a *app) unzipZipfile(zipfile string) error {
	err := unzip(a.conf.Unpacking, filepath.Join(a.conf.Downloading, zipfile))
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Join(a.conf.Downloading, zipfile))
	if err != nil {
		return err
	}

	return nil
}
