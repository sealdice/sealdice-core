package dice

import (
	"io"
	"net/http"
)

func GetCloudContent(urls []string, etag string) (int, []byte, error) {
	client := &http.Client{}
	for _, url := range urls {
		req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
		if err != nil {
			return 0, nil, err
		}
		// req.Header.Add("Accept", "application/toml;application/json")
		if etag != "" {
			req.Header.Add("If-None-Match", etag)
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		switch resp.StatusCode {
		case http.StatusNotModified:
			_ = resp.Body.Close()
			return http.StatusNotModified, nil, nil
		case http.StatusOK:
			newData, err := io.ReadAll(resp.Body)
			if err != nil {
				return 0, nil, err
			}
			return http.StatusOK, newData, nil
		default:
			return resp.StatusCode, nil, nil
		}
	}
	return http.StatusInternalServerError, nil, nil
}
