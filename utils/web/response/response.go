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

var logger = log.NewCustomHelper(log.LOG_API, false, nil)

const (
	ERROR        = 400
	SUCCESS      = 200
	SUCCESS_TEXT = "获取成功"
)

func GetGenericErrorMsg(err error) string {
	return fmt.Sprintf("执行失败，原因为:%v ，请反馈开发者", err)
}

func Result(code int, data interface{}, msg string, c echo.Context) error {
	// 返回JSON响应
	err := c.JSON(http.StatusOK, Response{
		Code: code,
		Data: data,
		Msg:  msg,
	})
	if err != nil {
		logger.Debug("Error: ", err)
		return err
	}
	return nil
}

func NoAuth(message string, c echo.Context) error {
	err := c.JSON(http.StatusUnauthorized, Response{
		Code: 7,
		Data: nil,
		Msg:  message,
	})
	if err != nil {
		logger.Debug("Error: ", err)
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

// 必要性不强，大部分情况下，我们不需要返回所谓data，只需要展示给用户提示信息即可
// func FailWithDetailed(data interface{}, message string, c echo.Context) {
//	Result(ERROR, data, message, c)
// }
