package api

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alexmullins/zip"
	"github.com/labstack/echo/v4"
	"github.com/monaco-io/request"

	"sealdice-core/dice"
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
	token := c.Request().Header.Get("token")
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

var times = 0

func getGithubAvatar(c echo.Context) error {
	times++
	if times > 500 {
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
		log.Println(err)
	}
	if err := compressFile(filepath.Join(rootPath, "device.json"), "device.json", zipWriter); err != nil {
		log.Println(err)
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
			} else {
				// 默认为gocq
				relWorkDir = "extra/go-cqhttp-qq" + uid
			}
			fmt.Println(relWorkDir, i.RelWorkDir)
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

var timeout = 5 * time.Second

func checkNetworkHealth(c echo.Context) error {
	total := 5 // baidu, seal, sign, google, github
	var ok []string
	baiduOk := checkHTTPConnectivity([]string{"https://baidu.com"}, timeout)
	if baiduOk {
		ok = append(ok, "baidu")
	}
	sealOk := checkHTTPConnectivity(dice.BackendUrls, timeout)
	if sealOk {
		ok = append(ok, "seal")
	}
	signOk := checkHTTPConnectivity([]string{"https://sign.lagrangecore.org/api/sign/ping"}, timeout)
	if signOk {
		ok = append(ok, "sign")
	}
	googleOk := checkHTTPConnectivity([]string{"https://google.com"}, timeout)
	if googleOk {
		ok = append(ok, "google")
	}
	githubOk := checkHTTPConnectivity([]string{"https://github.com"}, timeout)
	if githubOk {
		ok = append(ok, "github")
	}
	return Success(&c, Response{
		"total":     total,
		"ok":        ok,
		"timestamp": time.Now().Unix(),
	})
}

func checkHTTPConnectivity(urls []string, timeout time.Duration) bool {
	client := http.Client{
		Timeout: timeout,
	}
	ok := false
	for _, url := range urls {
		_, err := client.Get(url)
		if err == nil {
			ok = true
			break
		}
	}
	return ok
}
