package api

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/labstack/echo/v4"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sealdice-core/dice"
	"strings"
)

func jsExec(c echo.Context) error {
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}
	if !myDice.JsEnable {
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

	waitRun := make(chan int, 1)

	var ret goja.Value
	myDice.JsPrinter.RecordStart()
	myDice.JsLoop.RunOnLoop(func(vm *goja.Runtime) {
		defer func() {
			// 防止崩掉进程
			if r := recover(); r != nil {
				//fmt.Println("xx", r.(goja.Exception))
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
	if !myDice.JsEnable {
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
	if !myDice.JsEnable {
		resp := c.JSON(200, map[string]interface{}{
			"result": false,
			"err":    "js扩展支持已关闭",
		})
		return resp
	}

	v := struct {
		Index int `json:"index"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		if v.Index >= 0 && v.Index < len(myDice.JsScriptList) {
			dice.JsDelete(myDice, myDice.JsScriptList[v.Index])
		}
	}

	return c.JSON(http.StatusOK, nil)
}

func jsReload(c echo.Context) error {
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	//myDice.JsLock.Lock()
	//defer myDice.JsLock.Unlock()
	myDice.JsInit()
	myDice.JsLoadScripts()
	return c.JSON(200, nil)
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

	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	//-----------
	// Read file
	//-----------

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
	//fmt.Println("????", filepath.Join("./data/decks", file.Filename))
	file.Filename = strings.ReplaceAll(file.Filename, "/", "_")
	file.Filename = strings.ReplaceAll(file.Filename, "\\", "_")
	fmt.Println("XXXX", filepath.Join(myDice.BaseConfig.DataDir, "scripts", file.Filename))
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
	if !myDice.JsEnable {
		resp := c.JSON(200, []*dice.JsScriptInfo{})
		return resp
	}

	return c.JSON(http.StatusOK, myDice.JsScriptList)
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

	if myDice.JsEnable {
		myDice.JsShutdown()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"result": true,
	})
}

func jsStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"result": true,
		"status": myDice.JsEnable,
	})
}
