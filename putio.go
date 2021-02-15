package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	goputio "github.com/putdotio/go-putio"
)

func (a *app) fetchRemoteFile(ctx context.Context, file goputio.File) (string, error) {
	zipfile := file.Name + ".zip"

	zipCtx, zipCancelFn := context.WithTimeout(ctx, time.Minute)
	defer zipCancelFn()
	zip, err := a.client.RequestZip(zipCtx, file)
	if err != nil {
		return "", err
	}

	a.logger.Info("downloading file", "file", file.Name, "ID", file.ID)
	err = downloadFile(filepath.Join(a.conf.Downloading, zipfile), zip.URL)
	if err != nil {
		return "", err
	}

	a.logger.Info("deleting file", "file", file.Name, "ID", file.ID)
	deleteCtx, deleteCancelFn := context.WithTimeout(ctx, time.Minute)
	defer deleteCancelFn()
	err = a.client.DeleteFile(deleteCtx, file.ID)
	if err != nil {
		return "", err
	}

	return zipfile, nil
}

func (a *app) unzipZipfile(zipfile string) error {
	sourcePath := filepath.Join(a.conf.Downloading, zipfile)
	destPath := filepath.Join(a.conf.Unpacking, strings.TrimSuffix(zipfile, ".zip"))

	a.logger.Info("unzipping file", "source", sourcePath, "dest", destPath)
	err := unzip(destPath, sourcePath)
	if err != nil {
		return err
	}

	a.logger.Info("removing zip file", "path", sourcePath)
	err = os.Remove(sourcePath)
	if err != nil {
		return err
	}

	return nil
}
