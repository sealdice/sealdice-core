package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// 添加 checklist
func checklistAdd(c echo.Context) error { //nolint
	// var err error
	// cl := model.CheckList{}
	// db := model.GetDB()

	// if err := c.Bind(cl); err != nil {
	// 	return err
	// }

	// err = db.Create(&cl).Error
	// if err != nil {
	// 	return c.String(http.StatusBadRequest, "")
	// }

	return c.JSON(http.StatusCreated, nil)
}
