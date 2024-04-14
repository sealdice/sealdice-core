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

const uiResourcesURL = "https://github.com/sealdice/sealdice-ui/releases/download/pre-release/sealdice-ui.zip"

func main() {
	defer func() {
		_, err := os.Stat("sealdice-ui.zip")
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			panic("清理ui资源失败: " + err.Error())
		}
		err = os.Remove("sealdice-ui.zip")
		if err != nil {
			panic("清理ui资源失败: " + err.Error())
		}
	}()
	err := downloadFrontendZip()
	if err != nil {
		panic("下载ui资源失败: " + err.Error())
	}
	err = unzip("sealdice-ui.zip", "frontend")
	if err != nil {
		panic("解压ui资源失败: " + err.Error())
	}
}

func downloadFrontendZip() error {
	resp, err := http.Get(uiResourcesURL)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(resp.Status)
	}
	defer func() { _ = resp.Body.Close() }()

	file, err := os.Create("sealdice-ui.zip")
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Unzip 解压zip文件
// copy from https://stackoverflow.com/a/24792688
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	_ = os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name) //nolint:gosec

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(path, f.Mode())
		} else {
			_ = os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if errClose := f.Close(); errClose != nil {
					panic(errClose)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
