package api

import (
	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

func verifyGenerateCode(c echo.Context) error {
	mode := c.QueryParam("mode")
	var code string
	if mode == "base64" {
		code = dice.GenerateVerificationCode("UI", "UI:1001", "User", true)
	} else {
		code = dice.GenerateVerificationCode("UI", "UI:1001", "User", false)
	}
	if len(code) > 0 {
		return Success(&c, Response{
			"data": code,
		})
	} else {
		return Error(&c, "校验码生成失败", Response{})
	}
}
