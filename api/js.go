package api

import (
	"github.com/dop251/goja"
	"github.com/labstack/echo/v4"
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

	myDice.JsPrinter.RecordStart()
	source := "(function(exports, require, module) {" + v.Value + "\n})()"

	var ret goja.Value
	//myDice.JsLoop.Run(func(vm *goja.Runtime) {
	ret, err = myDice.JsVM.RunString(source)
	//})
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
