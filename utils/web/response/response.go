package response

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	log "sealdice-core/utils/kratos"
)

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

// 这里本来是初始化了一个自定义logger，但是好像会空指针，所以暂时用全局变量替代。

const (
	ERROR        = 400
	SUCCESS      = 200
	SUCCESS_TEXT = "获取成功"
)

func GetGenericErrorMsg(err error) string {
	return fmt.Sprintf("执行失败，原因为:%v ，请反馈开发者", err.Error())
}

func Result(code int, data interface{}, msg string, c echo.Context) error {
	// 返回JSON响应
	err := c.JSON(http.StatusOK, Response{
		Code: code,
		Data: data,
		Msg:  msg,
	})
	if err != nil {
		log.Debug("Error: ", err)
		return err
	}
	return nil
}

func NoAuth(c echo.Context) error {
	err := c.JSON(http.StatusUnauthorized, Response{
		Code: http.StatusUnauthorized,
		Data: nil,
		Msg:  "您未登录或登录态已失效",
	})
	if err != nil {
		log.Debug("Error: ", err)
		return err
	}
	return nil
}

func Ok(c echo.Context) error {
	return Result(SUCCESS, map[string]interface{}{}, "操作成功", c)
}

func OkWithMessage(message string, c echo.Context) error {
	return Result(SUCCESS, map[string]interface{}{}, message, c)
}

func OkWithData(data interface{}, c echo.Context) error {
	return Result(SUCCESS, data, SUCCESS_TEXT, c)
}

func OkWithDetailed(data interface{}, message string, c echo.Context) error {
	return Result(SUCCESS, data, message, c)
}

func Fail(c echo.Context) error {
	return Result(ERROR, map[string]interface{}{}, "操作失败", c)
}

func FailWithMessage(message string, c echo.Context) error {
	return Result(ERROR, map[string]interface{}{}, message, c)
}
