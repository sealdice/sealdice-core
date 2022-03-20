package dice

import (
	"bytes"
	"encoding/json"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"regexp"
)

func RemoveSpace(s string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, "")
}

func UploadFileToTransferSh(log *zap.SugaredLogger, filename string, data io.Reader) string {
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
func UploadFileToFileIo(log *zap.SugaredLogger, filename string, data io.Reader) string {
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

func JsonValueMapUnmarshal(data []byte, v *map[string]*VMValue) error {
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
					(*v)[key] = &VMValue{VMTypeInt64, i}
					continue
				}
			}
		}
	}
	return err
}

func GetRandomFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

var SVG_ICON = []byte(`<svg id="Layer_1" data-name="Layer 1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><defs><style>.cls-1{fill:#bdccd4;}.cls-2{fill:#fff;}</style></defs><title>icon-s</title><rect class="cls-1" x="6.36" y="7.14" width="498.86" height="498.86" rx="12"/><polygon class="cls-2" points="130.65 58.2 31.9 269.01 183.55 462.82 365.95 427.73 480.24 237.52 336.41 52 130.65 58.2"/><path d="M488.44,230.71,346.28,44.41h0a10.57,10.57,0,0,0-8.87-4.18L133.26,48.62h0a10.61,10.61,0,0,0-9.15,6L22.06,263h0a10.57,10.57,0,0,0,1.14,11.14l150.59,194.6a10.58,10.58,0,0,0,10.53,3.95L373.17,436l1.35-.46a10.59,10.59,0,0,0,5.72-4.71L489.16,242.44a10.6,10.6,0,0,0-.72-11.73ZM186,449.75l-24.1-187.3L385.9,376ZM364.21,107.21,159.67,244,140.55,72.87ZM149.65,248.53l-102.77,12L131.65,87.9ZM392.46,367.38,165.87,252.6l207-138.45,1.2,1.54,18.83,250.94ZM358.79,95.63,178.51,67.86,333,61.44ZM47.71,271.08l103.1-12L173.2,433.22ZM364.32,416l-120,23.36,135.29-49.82Zm38.14-65.88-16.62-219L467.21,238Z"/><polygon class="cls-2" points="157.03 220.4 160.14 249.69 178.19 258.84 374.4 120.32 373.55 108.64 358.48 106.33 157.03 220.4"/><path d="M297.84,193.19h0c-11-3.95-22.25-3.44-29.35,1.3-7.73,3.69-13.91,13-16.18,24.53C249,235.69,255,250.45,266,252.61c9.44,1.87,19.48-6.76,24-20.27,8.76,1.95,17,1,22.68-2.23a15,15,0,0,0,7-7.93C323.42,211.65,313.84,198.92,297.84,193.19Z"/><path d="M221.27,164c-8.94-3.2-18.77-2.18-27.68,2.88l-.08,0a44.16,44.16,0,0,0-19.37,23.68c-7.61,21.25,1.15,43.9,19.53,50.47,8.94,3.2,18.77,2.18,27.68-2.88l.08,0A44.16,44.16,0,0,0,240.8,214.5C248.41,193.25,239.65,170.61,221.27,164Z"/><ellipse class="cls-2" cx="194.6" cy="193" rx="21.33" ry="16.31" transform="translate(-62.71 287.4) rotate(-64.91)"/><circle class="cls-2" cx="225.91" cy="185.74" r="9.96"/><path d="M310.56,113.25a44.14,44.14,0,0,0-30.26,4.47,32.67,32.67,0,0,0-16.76,22.33c-3.78,19.15,11.16,38.29,33.3,42.66a44.15,44.15,0,0,0,30.26-4.47l.08-.05c8.92-5.06,14.84-13,16.68-22.28C347.64,136.76,332.7,117.62,310.56,113.25Z"/><ellipse class="cls-2" cx="286.98" cy="140.6" rx="21.33" ry="16.31" transform="translate(37.95 340.88) rotate(-64.91)"/><circle class="cls-2" cx="320.22" cy="132.25" r="9.96"/><ellipse cx="226.67" cy="154.45" rx="6.5" ry="9.75" transform="translate(-7.12 297.91) rotate(-65.8)"/><ellipse cx="252.33" cy="140.56" rx="9.75" ry="6.5" transform="translate(83.38 374.84) rotate(-83.32)"/></svg>`)
