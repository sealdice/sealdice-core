package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"sealdice-core/core"
)

func RemoveSpace(s string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, "")
}

func UploadFileToTransferSh(filename string, data io.Reader) string {
	log := core.GetLogger()
	client := &http.Client{}
	req, err := http.NewRequest("PUT", "https://transfer.sh/" + filename, data)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("authority", "transfer.sh")
	req.Header.Set("content-length", "1129")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.192 Safari/537.36 OPR/74.0.3911.218")
	req.Header.Set("content-type", "application/x-zip-compressed")
	req.Header.Set("accept", "*/*")
	req.Header.Set("origin", "https://transfer.sh")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("referer", "https://transfer.sh/")
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(bodyText) // 返回url地址
}
