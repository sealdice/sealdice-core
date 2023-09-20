package api

import (
	"github.com/labstack/echo/v4"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"runtime"
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

	form, err := c.MultipartForm()
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	files := form.File["files"]
	if len(files) > 0 {
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
		f2, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return Error(&c, err.Error(), Response{})
		}
		_, err = io.Copy(f2, src)
		if err != nil {
			return Error(&c, err.Error(), Response{})
		}

		if dm.UpdateSealdiceByFile != nil {
			if dm.UpdateSealdiceByFile(fn) {
				return Success(&c, Response{"result": true})
			} else {
				return Error(&c, "自动升级流程失败，请检查控制台输出", Response{})
			}
		} else {
			return Error(&c, "骰子没有正确初始化，无法使用此功能", Response{})
		}
	}

	return Error(&c, "参数错误", Response{})
}
