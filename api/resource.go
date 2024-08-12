package api

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sunshineplan/imgconv"
)

type resourceType string

const (
	Image   resourceType = "image"
	Unknown resourceType = "unknown"
)

type resource struct {
	Type resourceType `json:"type"`
	Name string       `json:"name"`
	Ext  string       `json:"ext"`
	Path string       `json:"path"`
	Size int64        `json:"size"`
}

func resourceGetList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}

	// 目前只支持图片
	var images []resource
	_ = filepath.Walk("data/images", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			t, ext := getResourceType(path)
			if t == Image {
				images = append(images, resource{
					Type: t,
					Name: filepath.Base(path),
					Ext:  ext,
					Path: filepath.ToSlash(path),
					Size: info.Size(),
				})
			}
		}
		return nil
	})

	return Success(&c, map[string]interface{}{
		"data": images,
	})
}

func getResourceType(filename string) (resourceType, string) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif":
		return Image, ext
	default:
		return Unknown, ext
	}
}

type resourceOperationParam struct {
	Path string `query:"path"`
}

func resourceDownload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}
	var param resourceOperationParam
	err := c.Bind(&param)
	param.Path = filepath.FromSlash(param.Path)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	err = ensurePathSafe(param.Path)
	if err != nil {
		return c.NoContent(http.StatusForbidden)
	}

	res, err := os.Stat(param.Path)
	if err != nil {
		return c.NoContent(http.StatusNotFound)
	}
	return c.Attachment(param.Path, res.Name())
}

func resourceUpload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	files := form.File["files"]
	if len(files) == 0 {
		return Error(&c, "请上传文件", Response{})
	}

	cwd, _ := os.Getwd()
	var errors []string
	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			errors = append(errors, file.Filename)
			continue
		}
		defer func(src multipart.File) {
			_ = src.Close()
		}(src)

		file.Filename = strings.ReplaceAll(file.Filename, "/", "_")
		file.Filename = strings.ReplaceAll(file.Filename, "\\", "_")

		resType, _ := getResourceType(file.Filename)
		var path string
		if resType == Image {
			path = filepath.Join(cwd, "data/images", filepath.Base(file.Filename))
		} else {
			errors = append(errors, file.Filename)
			myDice.Logger.Errorf("保存资源文件「%s」失败，暂时不支持这种类型", path)
			continue
		}

		myDice.Logger.Infof("保存资源文件: %s", path)
		var dst *os.File
		dst, err = os.Create(path)
		if err != nil {
			errors = append(errors, file.Filename)
			continue
		}
		defer func(dst *os.File) {
			_ = dst.Close()
		}(dst)

		if _, err = io.Copy(dst, src); err != nil {
			errors = append(errors, file.Filename)
			continue
		}
	}

	if len(errors) > 0 {
		return Error(&c, "部分文件上传失败：\n"+strings.Join(errors, "\n"), Response{})
	}

	return Success(&c, Response{})
}

func resourceDelete(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}
	var param resourceOperationParam
	err := c.Bind(&param)
	param.Path = filepath.FromSlash(param.Path)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	err = ensurePathSafe(param.Path)
	if err != nil {
		return c.NoContent(http.StatusForbidden)
	}

	err = os.Remove(param.Path)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	return Success(&c, Response{})
}

type resourceDataParam struct {
	Path      string `query:"path"`
	Thumbnail bool   `query:"thumbnail"`
}

func resourceGetData(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	var param resourceDataParam
	err := c.Bind(&param)
	param.Path = filepath.FromSlash(param.Path)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	err = ensurePathSafe(param.Path)
	if err != nil {
		return c.NoContent(http.StatusForbidden)
	}

	img, _ := imgconv.Open(param.Path)
	if param.Thumbnail {
		img = imgconv.Resize(img, &imgconv.ResizeOption{Width: 64})
	}

	buf := new(bytes.Buffer)
	_ = imgconv.Write(buf, img, &imgconv.FormatOption{Format: imgconv.PNG})

	return c.Blob(http.StatusOK, "", buf.Bytes())
}

func ensurePathSafe(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	imagesPath := filepath.Join(cwd, "data/images")
	audiosPath := filepath.Join(cwd, "data/audios")
	videosPath := filepath.Join(cwd, "data/videos")

	if !strings.HasPrefix(abs, imagesPath) &&
		!strings.HasPrefix(abs, audiosPath) &&
		!strings.HasPrefix(abs, videosPath) {
		return fmt.Errorf("invalid path")
	}
	return nil
}
