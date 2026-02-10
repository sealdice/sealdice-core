package api

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

func jsExec(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}
	if !myDice.Config.JsEnable {
		resp := c.JSON(200, map[string]interface{}{
			"result": false,
			"err":    "js扩展支持已关闭",
		})
		return resp
	}

	v := struct {
		Value string `json:"value"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}

	source := "(function(exports, require, module) {" + v.Value + "\n})()"
	loop := myDice.ExtLoopManager.GetWebLoop()
	waitRun := make(chan int, 1)

	var ret goja.Value
	myDice.JsPrinter.RecordStart()
	loop.RunOnLoop(func(vm *goja.Runtime) {
		defer func() {
			// 防止崩掉进程
			if r := recover(); r != nil {
				// fmt.Println("xx", r.(goja.Exception))
				myDice.JsPrinter.Error(fmt.Sprintf("JS脚本报错: %v", r))
			}
			waitRun <- 1
		}()
		ret, err = vm.RunString(source)
	})
	<-waitRun
	outputs := myDice.JsPrinter.RecordEnd()

	var retFinal interface{}
	if ret != nil {
		retFinal = ret.Export()
	}

	var errText interface{}
	if err != nil {
		errText = err.Error()
	}

	resp := c.JSON(200, map[string]interface{}{
		"result":  true,
		"ret":     retFinal,
		"outputs": outputs,
		"err":     errText,
	})

	return resp
}

func jsGetRecord(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if !myDice.Config.JsEnable {
		resp := c.JSON(200, map[string]interface{}{
			"outputs": []string{},
		})
		return resp
	}

	outputs := myDice.JsPrinter.RecordEnd()
	resp := c.JSON(200, map[string]interface{}{
		"outputs": outputs,
	})
	return resp
}

func jsDelete(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}
	if !myDice.Config.JsEnable {
		resp := c.JSON(200, map[string]interface{}{
			"result": false,
			"err":    "js扩展支持已关闭",
		})
		return resp
	}

	v := struct {
		Filename string `json:"filename"`
	}{}
	err := c.Bind(&v)

	if err == nil && v.Filename != "" {
		for _, js := range myDice.JsScriptList {
			if js.Filename == v.Filename {
				dice.JsDelete(myDice, js)
				break
			}
		}
	}

	return c.JSON(http.StatusOK, nil)
}

func jsReload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}
	// 尝试取锁，如果取不到，说明正在后台重载中
	// TODO:用户提示模式？
	locked := myDice.JsReloadLock.TryLock()
	if !locked {
		return c.NoContent(400)
	}
	defer myDice.JsReloadLock.Unlock()
	myDice.JsReload()
	return c.NoContent(200)
}

func jsUpload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	// -----------
	// Read file
	// -----------

	// Source
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func(src multipart.File) {
		_ = src.Close()
	}(src)

	// Destination
	// fmt.Println("????", filepath.Join("./data/decks", file.Filename))
	file.Filename = strings.ReplaceAll(file.Filename, "/", "_")
	file.Filename = strings.ReplaceAll(file.Filename, "\\", "_")
	dst, err := os.Create(filepath.Join(myDice.BaseConfig.DataDir, "scripts", file.Filename))
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		_ = dst.Close()
	}(dst)

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, nil)
}

func jsList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if !myDice.Config.JsEnable {
		resp := c.JSON(200, []*dice.JsScriptInfo{})
		return resp
	}

	type script struct {
		dice.JsScriptInfo
		BuiltinUpdated bool `json:"builtinUpdated"`
	}
	scripts := make([]*script, 0, len(myDice.JsScriptList))
	for _, info := range myDice.JsScriptList {
		temp := script{
			JsScriptInfo:   *info,
			BuiltinUpdated: info.Builtin && !myDice.JsBuiltinDigestSet[info.Digest],
		}
		scripts = append(scripts, &temp)
	}

	return c.JSON(http.StatusOK, scripts)
}

func jsShutdown(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	if myDice.Config.JsEnable {
		myDice.JsShutdown()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"result": true,
	})
}

func jsStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"result": true,
		"status": myDice.Config.JsEnable,
	})
}

func jsEnable(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}
	v := struct {
		Name string `form:"name" json:"name"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		dice.JsEnable(myDice, v.Name)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"result": true,
			"name":   v.Name,
		})
	}
	return c.JSON(http.StatusBadRequest, nil)
}

func jsDisable(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}
	v := struct {
		Name string `form:"name" json:"name"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		dice.JsDisable(myDice, v.Name)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"result": true,
			"name":   v.Name,
		})
	}

	return c.JSON(http.StatusBadRequest, nil)
}

func jsCheckUpdate(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}
	v := struct {
		Filename string `json:"filename"`
	}{}
	err := c.Bind(&v)

	if err == nil && v.Filename != "" {
		for _, jsScript := range myDice.JsScriptList {
			if jsScript.Filename == v.Filename {
				oldJs, newJs, tempFileName, errUpdate := myDice.JsCheckUpdate(jsScript)
				if errUpdate != nil {
					return Error(&c, errUpdate.Error(), Response{})
				}
				return Success(&c, Response{
					"old":          oldJs,
					"new":          newJs,
					"format":       "javascript",
					"filename":     jsScript.Filename,
					"tempFileName": tempFileName,
				})
			}
		}
		return Error(&c, "未找到脚本", Response{})
	}
	return Success(&c, Response{})
}

func jsUpdate(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}
	if !myDice.Config.JsEnable {
		return Error(&c, "js扩展支持已关闭", Response{})
	}

	v := struct {
		Filename     string `json:"filename"`
		TempFileName string `json:"tempFileName"`
	}{}
	err := c.Bind(&v)

	if err == nil && v.Filename != "" {
		for _, jsScript := range myDice.JsScriptList {
			if jsScript.Filename == v.Filename {
				err = myDice.JsUpdate(jsScript, v.TempFileName)
				if err != nil {
					return Error(&c, err.Error(), Response{})
				}
				myDice.MarkModified()
				break
			}
		}
	}
	return Success(&c, Response{})
}
