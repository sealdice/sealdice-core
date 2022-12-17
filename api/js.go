package api

import (
	"github.com/dop251/goja"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func jsExec(c echo.Context) error {
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Value string `json:"value"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}

	myDice.JsLock.Lock()
	defer func() {
		myDice.JsLock.Unlock()
	}()

	source := "(function(exports, require, module) {" + v.Value + "\n})()"

	var ret goja.Value
	myDice.JsPrinter.RecordStart()
	myDice.JsLoop.Run(func(vm *goja.Runtime) {
		ret, err = vm.RunString(source)
	})
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
		"ret":     retFinal,
		"outputs": outputs,
		"err":     errText,
	})

	return resp
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
	defer src.Close()

	// Destination
	//fmt.Println("????", filepath.Join("./data/decks", file.Filename))
	file.Filename = strings.ReplaceAll(file.Filename, "/", "_")
	file.Filename = strings.ReplaceAll(file.Filename, "\\", "_")
	dst, err := os.Create(filepath.Join(myDice.BaseConfig.DataDir, "scripts", file.Filename))
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, nil)
}
