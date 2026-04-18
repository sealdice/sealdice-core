package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexmullins/zip"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"

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
	token := c.Request().Header.Get("token") //nolint:canonicalheader // private header
	if token == "" {
		token = c.QueryParam("token")
	}
	return myDice.Parent.AccessTokens.Exists(token)
}

func GetHexData(c echo.Context, method string, name string) (value []byte, finished bool) {
	var err error
	var strValue string
	// var exists bool

	switch method {
	case "GET":
		strValue = c.Param(name)
	case "POST":
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

func getGithubAvatar(c echo.Context) error {
	// 因为不明原因阻塞，会导致显示离线，先注释掉了
	uid := c.Param("uid")
	return c.Redirect(http.StatusFound, fmt.Sprintf("https://avatars.githubusercontent.com/%s?s=200", uid))

	// uid := c.Param("uid")
	// req := request.Client{
	// 	URL:    fmt.Sprintf("https://avatars.githubusercontent.com/%s?s=200", uid),
	// 	Method: "GET",
	// }

	// resp := req.Send()
	// if resp.OK() {
	// 	// 设置缓存时间为3天
	// 	c.Response().Header().Set("Cache-Control", "max-age=259200")

	// 	return c.Blob(http.StatusOK, resp.ContentType(), resp.Bytes())
	// }
	// return c.JSON(http.StatusNotFound, "")
}

func packGocqConfig(relWorkDir string) *bytes.Buffer {
	// workDir := "extra/go-cqhttp-qq" + account
	rootPath := filepath.Join(myDice.BaseConfig.DataDir, relWorkDir)

	// 创建一个内存缓冲区，用于保存 Zip 文件内容
	buf := new(bytes.Buffer)

	// 创建 Zip Writer，将 Zip 文件内容写入内存缓冲区
	zipWriter := zip.NewWriter(buf)

	if err := compressFile(filepath.Join(rootPath, "config.yml"), "config.yml", zipWriter); err != nil {
		myDice.Logger.Error(err)
	}
	if err := compressFile(filepath.Join(rootPath, "device.json"), "device.json", zipWriter); err != nil {
		myDice.Logger.Error(err)
	}
	_ = compressFile(filepath.Join(rootPath, "data/versions/1.json"), "data/versions/6.json", zipWriter)
	_ = compressFile(filepath.Join(rootPath, "data/versions/6.json"), "data/versions/6.json", zipWriter)

	// 关闭 Zip Writer
	if err := zipWriter.Close(); err != nil {
		myDice.Logger.Fatal(err)
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
			switch pa.BuiltinMode {
			case "lagrange":
				relWorkDir = "extra/lagrange-qq" + uid
			case "lagrange-gocq":
				relWorkDir = "extra/lagrange-gocq-qq" + uid
			default:
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
	checkTimes                 = 3
	checkTimeout time.Duration = 5 * time.Second
)

func checkHTTPConnectivity(url string) (bool, time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	type rs struct {
		ok       bool
		duration time.Duration
	}
	rsChan := make(chan rs, checkTimes)
	once := func(url string) {
		myDice.Logger.Debugf("check http connectivity, url=%s", url)
		start := time.Now()
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := http.DefaultClient.Do(req) //nolint:gosec
		duration := time.Since(start)
		if err == nil {
			_ = resp.Body.Close()
			rsChan <- rs{true, duration}
		} else {
			myDice.Logger.Debugf("url can't be connected, error: %s", err)
			rsChan <- rs{false, duration}
		}
	}

	var wg sync.WaitGroup
	for range checkTimes {
		// wg.Go(func() { once(url) }) // 1.25写法，但是编译后无法正常运行
		wg.Add(1)
		go func() {
			defer wg.Done()
			once(url)
		}()
	}
	wg.Wait()
	close(rsChan)

	ok := true
	var (
		totalDuration int64 = 0
		count         int64 = 0
		duration      int64 = 0
	)
	for res := range rsChan {
		ok = ok && res.ok
		if res.ok {
			count++
			totalDuration += int64(res.duration)
		}
	}
	if count != 0 {
		duration = totalDuration / count
	}
	return ok, time.Duration(duration)
}

func checkNetworkHealth(c echo.Context) error {
	total := 5 // baidu, seal, sign, google, github
	var wg sync.WaitGroup

	type rs struct {
		Target   string        `json:"target"`
		Ok       bool          `json:"ok"`
		Duration time.Duration `json:"duration"`
	}
	rsChan := make(chan rs, total)

	checkUrls := func(target string, urls []string) {
		for _, url := range urls {
			ok, duration := checkHTTPConnectivity(url)
			if ok {
				rsChan <- rs{
					Target:   target,
					Ok:       true,
					Duration: duration,
				}
				return
			}
		}
		rsChan <- rs{
			Target:   target,
			Ok:       false,
			Duration: 0,
		}
	}

	signGroups, err := dice.LagrangeGetSignInfo(myDice)
	if err == nil && len(signGroups) > 0 {
		signServers := signGroups[len(signGroups)-1].Servers // 取下发列表中 version 最新的签名服务器组，即最后一条
		urls := lo.Map(signServers, func(signServerInfo *dice.SignServerInfo, _ int) string {
			ping, _ := url.JoinPath(signServerInfo.Url, "/ping")
			return ping
		})
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkUrls("sign", urls)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		checkUrls("baidu", []string{"https://baidu.com"})
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		checkUrls("seal", dice.BackendUrls)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		checkUrls("google", []string{"https://google.com"})
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		checkUrls("github", []string{"https://github.com"})
	}()

	wg.Wait()
	close(rsChan)

	var ok []string
	var targets []rs
	for target := range rsChan {
		targets = append(targets, target)
		if target.Ok {
			ok = append(ok, target.Target)
		}
	}

	return Success(&c, Response{
		"total":     total,
		"ok":        ok, // 被 targets 代替，可废弃，但先为接口兼容保留
		"targets":   targets,
		"timestamp": time.Now().Unix(),
	})
}
