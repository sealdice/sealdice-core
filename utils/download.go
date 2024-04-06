package utils

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func DownloadFile(filepath string, url string) error {
	// Get the data
	// resp, err := http.Get(url)
	client := new(http.Client)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	request.Header.Add("Accept-Encoding", "gzip")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	if resp.StatusCode == http.StatusOK {
		// Write the body to file
		if resp.Header.Get("Content-Encoding") == "gzip" {
			// 如果响应使用了GZIP压缩，需要解压缩
			var reader io.ReadCloser
			reader, err = gzip.NewReader(resp.Body)
			if err != nil {
				fmt.Println("GZIP解压出错:", err)
				return err
			}
			defer reader.Close()
			_, err = io.Copy(out, reader)
		} else {
			_, err = io.Copy(out, resp.Body)
		}

		return err
	}

	return errors.New("http status:" + resp.Status)
}
