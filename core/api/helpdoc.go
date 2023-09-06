package api

import (
	"mime/multipart"
	"net/http"
	"sealdice-core/dice"

	"github.com/labstack/echo/v4"
)

func helpDocStatus(c echo.Context) error {
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}
	return Success(&c, Response{
		"loading": dm.IsHelpReloading,
	})
}

func helpDocTree(c echo.Context) error {
	if !doAuth(c) {
		return c.NoContent(http.StatusForbidden)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}

	if !dm.IsHelpReloading {
		return Success(&c, Response{
			"data": dm.Help.HelpDocTree,
		})
	} else {
		return Error(&c, "帮助文件正在加载", Response{})
	}
}

func helpDocReload(c echo.Context) error {
	if !doAuth(c) {
		return c.NoContent(http.StatusForbidden)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}

	if !dm.IsHelpReloading {
		dm.IsHelpReloading = true
		dm.Help.Close()

		dm.InitHelp()
		dm.AddHelpWithDice(dm.Dice[0])
		return Success(&c, Response{})
	} else {
		return Error(&c, "帮助文档正在重新装载", Response{})
	}
}

func helpDocUpload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Success(&c, Response{
			"testMode": true,
		})
	}

	group := c.FormValue("group")
	form, err := c.MultipartForm()
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	if group == "builtin" {
		return Error(&c, "不能为内置分组", Response{})
	}
	files := form.File["files"]
	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			return Error(&c, err.Error(), Response{})
		}
		defer func(src multipart.File) {
			_ = src.Close()
		}(src)

		err = dm.Help.UploadHelpDoc(src, group, file.Filename)
		if err != nil {
			return Error(&c, "上传文件 "+file.Filename+" 失败："+err.Error(), Response{})
		}
	}

	return Success(&c, Response{"result": true})
}

func helpDocDelete(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Success(&c, Response{
			"testMode": true,
		})
	}

	v := struct {
		Keys []string `json:"keys"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		err := dm.Help.DeleteHelpDoc(v.Keys)
		if err != nil {
			return Error(&c, err.Error(), Response{})
		}
	}

	return Success(&c, Response{})
}

func helpGetTextItemPage(c echo.Context) error {
	v := struct {
		PageNum  int    `json:"pageNum"`
		PageSize int    `json:"pageSize"`
		Id       string `json:"id"`
		Group    string `json:"group"`
		From     string `json:"from"`
		Title    string `json:"title"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		total, data := dm.Help.GetHelpItemPage(v.PageNum, v.PageSize, v.Id, v.Group, v.From, v.Title)
		return Success(&c, Response{"total": total, "data": data})
	}
	return Success(&c, Response{"total": 0, "data": dice.HelpTextItems{}})
}
