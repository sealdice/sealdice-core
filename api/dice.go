package api

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"runtime"

	"github.com/labstack/echo/v4"
)

func DiceNewVersionUpload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Success(&c, Response{
			"testMode": true,
		})
	}

	if dm.UpdateSealdiceByFile == nil {
		return Error(&c, "骰子没有正确初始化，无法使用此功能", Response{})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	files := form.File["files"]
	if len(files) == 0 {
		return Error(&c, "参数错误", Response{})
	}

	myDice.Logger.Info("收到新版本骰子上传请求")

	file := files[0]
	src, err := file.Open()
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	defer func(src multipart.File) {
		_ = src.Close()
	}(src)

	// TODO: 临时将逻辑写在这里，后续根据情况再调整
	fn := "./new_package"
	if runtime.GOOS == "windows" {
		fn += ".zip"
	} else {
		fn += ".tar.gz"
	}

	f2, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	_, err = io.Copy(f2, src)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	f2.Close()

	myDice.Logger.Info("新版本骰子上传成功")

	_ = Success(&c, Response{"result": true})

	if !dm.UpdateSealdiceByFile(fn) {
		myDice.Logger.Error("更新骰子失败")
	}

	return nil
}
