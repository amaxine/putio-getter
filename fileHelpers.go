package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func downloadFile(dst string, src string) error {
	resp, err := http.Get(src)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(dst string, src string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fp := filepath.Join(dst, f.Name)
		// https://snyk.io/research/zip-slip-vulnerability#go
		if !strings.HasPrefix(fp, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", f.Name)
		}

		if f.FileInfo().IsDir() {
			continue
		}

		err = os.MkdirAll(filepath.Dir(fp), os.ModePerm)
		if err != nil {
			return err
		}

		out, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		in, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}

		_, err = io.Copy(out, in)

		out.Close()
		in.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
