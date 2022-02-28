package dice

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
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
	req, err := http.NewRequest("PUT", "https://transfer.sh/"+filename, data)
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

// fileio 似乎会被风控
func UploadFileToFileIo(filename string, data io.Reader) string {
	log := core.GetLogger()
	client := &http.Client{}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filename)
	io.Copy(part, data)
	writer.Close()

	req, err := http.NewRequest("POST", "https://file.io/", body)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("authority", "file.io")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.192 Safari/537.36 OPR/74.0.3911.218")
	req.Header.Set("content-type", writer.FormDataContentType())
	//req.Header.Set("content-type", "multipart/form-data; boundary=----WebKitFormBoundaryANywTSoLmaYriaWC")
	req.Header.Set("origin", "https://www.file.io")
	req.Header.Set("sec-fetch-site", "same-site")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("referer", "https://www.file.io/")
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

	var ret struct {
		Link string `json:"string"`
	}
	_ = json.Unmarshal(bodyText, &ret)
	return ret.Link

	//autoDelete: true
	//created: "2022-02-25T04:57:05.297Z"
	//downloads: 0
	//expires: "2022-03-11T04:57:05.297Z"
	//id: "60abdf20-95f7-11ec-90fb-93917ea04b9f"
	//key: "k2ZQDU0do6RV"
	//link: "https://file.io/k2ZQDU0do6RV"
	//maxDownloads: 1
	//mimeType: "application/x-zip-compressed"
	//modified: "2022-02-25T04:57:05.297Z"
	//Name: "2022_02_24_17_11_38.zip"
	//private: false
	//size: 1359
	//status: 200
	//Success: true
}

func JsonNumberUnmarshal(data []byte, v interface{}) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	return d.Decode(v)
}

func JsonValueMapUnmarshal(data []byte, v *map[string]VMValue) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	err := d.Decode(v)
	if err == nil {
		for key, val := range *v {
			if val.TypeId == VMTypeInt64 {
				n, ok := val.Value.(json.Number)
				if !ok {
					continue
				}
				if i, err := n.Int64(); err == nil {
					(*v)[key] = VMValue{VMTypeInt64, i}
					continue
				}
			}
		}
	}
	return err
}
