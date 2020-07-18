package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/putdotio/go-putio"
)

// Waits until prepared zip file is ready at putio
func (a *app) waitForZip(ctx context.Context, zipID int64) (*putio.Zip, error) {
	ticker := time.NewTimer(0)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			zip, err := a.client.Zips.Get(context.Background(), zipID)
			if err != nil {
				return nil, err
			}

			if zip.URL != "" {
				return &zip, nil
			}

			ticker.Reset(time.Second)
		}
	}
}

// Waits until deletion of remote file succeeds to avoid duplicate downloads
func (a *app) waitForDelete(ctx context.Context, fileID int64) error {
	ticker := time.NewTimer(0)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to delete file before timeout, %v", ctx.Err())
		case <-ticker.C:
			err := a.client.Files.Delete(context.Background(), fileID)
			if err != nil {
				log.Println(err)
				ticker.Reset(5 * time.Second)
				break
			}

			return nil
		}
	}
}

func (a *app) fetchRemoteFile(file putio.File) error {
	a.logger.Info("found " + file.Name)
	zipfile := file.Name + ".zip"
	zipID, err := a.client.Zips.Create(context.Background(), file.ID)
	if err != nil {
		return err
	}

	zipCtx, zipCancelFn := context.WithTimeout(context.TODO(), time.Minute)
	defer zipCancelFn()
	zip, err := a.waitForZip(zipCtx, zipID)
	if err != nil {
		return err
	}

	a.logger.Debug("Fetching file", "file", file.Name, "url", zip.URL)
	err = os.MkdirAll(a.conf.Downloading, os.ModePerm)
	if err != nil {
		return err
	}
	err = downloadFile(filepath.Join(a.conf.Downloading, zipfile), zip.URL)
	if err != nil {
		return err
	}

	a.logger.Debug("Finished downloading. Deleting file.")
	deleteCtx, deleteCancelFn := context.WithTimeout(context.TODO(), time.Minute)
	defer deleteCancelFn()
	err = a.waitForDelete(deleteCtx, file.ID)
	if err != nil {
		return err
	}

	a.logger.Debug("Unzipping file.")
	err = unzip(a.conf.Unpacking, filepath.Join(a.conf.Downloading, zipfile))
	if err != nil {
		return err
	}
	a.logger.Debug("Removing zip file.")
	err = os.Remove(filepath.Join(a.conf.Downloading, zipfile))
	if err != nil {
		return err
	}

	return nil
}
