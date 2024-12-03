package api

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexmullins/zip"
	"github.com/labstack/echo/v4"
	"github.com/monaco-io/request"

	"sealdice-core/dice"
	log "sealdice-core/utils/kratos"
)

type Response map[string]interface{}

func Success(c *echo.Context, res Response) error {
	res["result"] = true
	return (*c).JSON(http.StatusOK, res)
}

func Error(c *echo.Context, errMsg string, res Response) error {
	res["result"] = false
	res["err"] = errMsg
	return (*c).JSON(http.StatusOK, res)
}

func Int64ToBytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func doAuth(c echo.Context) bool {
	token := c.Request().Header.Get("token") //nolint:canonicalheader // private header
	if token == "" {
		token = c.QueryParam("token")
	}
	if myDice.Parent.AccessTokens[token] {
		return true
	}
	return false
}

func GetHexData(c echo.Context, method string, name string) (value []byte, finished bool) {
	var err error
	var strValue string
	// var exists bool

	if method == "GET" {
		strValue = c.Param(name)
	} else if method == "POST" {
		strValue = c.FormValue(name)
	}

	// if !exists {
	// 	c.String(http.StatusNotAcceptable, "")
	// 	return nil, true
	// }

	value, err = hex.DecodeString(strValue)
	if err != nil {
		_ = c.String(http.StatusBadRequest, "")
		return nil, true
	}

	return value, false
}

var getAvatarCounter = 0

func getGithubAvatar(c echo.Context) error {
	getAvatarCounter++
	if getAvatarCounter > 500 {
		// 请求次数过多
		return c.JSON(http.StatusNotFound, "")
	}

	uid := c.Param("uid")
	req := request.Client{
		URL:    fmt.Sprintf("https://avatars.githubusercontent.com/%s?s=200", uid),
		Method: "GET",
	}

	resp := req.Send()
	if resp.OK() {
		// 设置缓存时间为3天
		c.Response().Header().Set("Cache-Control", "max-age=259200")

		return c.Blob(http.StatusOK, resp.ContentType(), resp.Bytes())
	}
	return c.JSON(http.StatusNotFound, "")
}

func packGocqConfig(relWorkDir string) *bytes.Buffer {
	// workDir := "extra/go-cqhttp-qq" + account
	rootPath := filepath.Join(myDice.BaseConfig.DataDir, relWorkDir)

	// 创建一个内存缓冲区，用于保存 Zip 文件内容
	buf := new(bytes.Buffer)

	// 创建 Zip Writer，将 Zip 文件内容写入内存缓冲区
	zipWriter := zip.NewWriter(buf)

	if err := compressFile(filepath.Join(rootPath, "config.yml"), "config.yml", zipWriter); err != nil {
		log.Error(err)
	}
	if err := compressFile(filepath.Join(rootPath, "device.json"), "device.json", zipWriter); err != nil {
		log.Error(err)
	}
	_ = compressFile(filepath.Join(rootPath, "data/versions/1.json"), "data/versions/6.json", zipWriter)
	_ = compressFile(filepath.Join(rootPath, "data/versions/6.json"), "data/versions/6.json", zipWriter)

	// 关闭 Zip Writer
	if err := zipWriter.Close(); err != nil {
		log.Fatal(err)
	}

	// 将 Zip 文件保存在内存中
	return buf
}

func compressFile(fn string, zipFn string, zipWriter *zip.Writer) error {
	data, err := os.ReadFile(fn)
	if err != nil {
		return err
	}

	h := &zip.FileHeader{Name: zipFn, Method: zip.Deflate, Flags: 0x800}
	fileWriter, err := zipWriter.CreateHeader(h)
	if err != nil {
		return err
	}
	_, _ = fileWriter.Write(data)
	return nil
}

func checkUidExists(c echo.Context, uid string) bool {
	for _, i := range myDice.ImSession.EndPoints {
		if pa, ok := i.Adapter.(*dice.PlatformAdapterGocq); ok && pa.UseInPackClient {
			var relWorkDir string
			if pa.BuiltinMode == "lagrange" {
				relWorkDir = "extra/lagrange-qq" + uid
			} else if pa.BuiltinMode == "lagrange-gocq" {
				relWorkDir = "extra/lagrange-gocq-qq" + uid
			} else {
				// 默认为gocq
				relWorkDir = "extra/go-cqhttp-qq" + uid
			}
			if relWorkDir == i.RelWorkDir {
				// 不允许工作路径重复
				_ = c.JSON(CodeAlreadyExists, i)
				return true
			}
		}

		// 如果存在已经启用的同账号连接，不允许重复
		if i.Enable && i.UserID == dice.FormatDiceIDQQ(uid) {
			_ = c.JSON(CodeAlreadyExists, i)
			return true
		}
	}
	return false
}

const (
	checkTimes   = 3
	checkTimeout = 5 * time.Second
)

func checkHTTPConnectivity(url string) bool {
	client := http.Client{
		Timeout: checkTimeout,
	}
	rsChan := make(chan bool, checkTimes)
	once := func(wg *sync.WaitGroup, url string) {
		defer wg.Done()
		resp, err := client.Get(url)
		log.Debugf("check http connectivity, url=%s", url)
		if err == nil {
			_ = resp.Body.Close()
			rsChan <- true
		} else {
			log.Debugf("url can't be connected, error: %s", err)
			rsChan <- false
		}
	}

	var wg sync.WaitGroup
	wg.Add(checkTimes)
	for range checkTimes {
		go once(&wg, url)
	}
	go func() {
		wg.Wait()
		close(rsChan)
	}()

	ok := true
	for res := range rsChan {
		ok = ok && res
	}
	return ok
}

func checkNetworkHealth(c echo.Context) error {
	total := 5 // baidu, seal, sign, google, github
	var ok []string
	var wg sync.WaitGroup
	wg.Add(total)
	rsChan := make(chan string, 5)

	checkUrls := func(target string, urls []string) {
		defer wg.Done()
		for _, url := range urls {
			if checkHTTPConnectivity(url) {
				rsChan <- target
				break
			}
		}
	}
	go checkUrls("baidu", []string{"https://baidu.com"})
	go checkUrls("seal", dice.BackendUrls)
	go checkUrls("sign", []string{"https://sign.lagrangecore.org/api/sign/ping"})
	go checkUrls("google", []string{"https://google.com"})
	go checkUrls("github", []string{"https://github.com"})

	go func() {
		wg.Wait()
		close(rsChan)
	}()

	for targetOk := range rsChan {
		ok = append(ok, targetOk)
	}

	return Success(&c, Response{
		"total":     total,
		"ok":        ok,
		"timestamp": time.Now().Unix(),
	})
}
