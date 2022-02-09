package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"sealdice-core/api"
	"sealdice-core/core"
	"sealdice-core/model"
)


func main() {
	core.LoggerInit()
	model.DBInit()
	defer model.GetDB().Close()

	//args2, _ := flags.ParseArgs(&struct {}{}, args)
	//fmt.Println(args2, opts)
	//CommandParse("。asdasd")
	//CommandParse("。asdasd   222")
	//CommandParse(".aaa bbb ccc ddd")
	//CommandParse("  .aaa bbb ccc ddd")
	//CommandParse("  .aaa bbb \n   ccc ddd")
	//ArgsParse("aaaa -vv ffas -f=3")

	//AtParse("[CQ:at,qq=3604749540]")
	//AtParse("[CQ:at,qq=3604749540] 333 [CQ:at,qq=3604749540] 2222")
	//AtParse("[CQ:at,qq=3604749540] 333 \n[CQ:at,qq=3604749540] 2222")

	dice := &Dice{};
	dice.init();

	dice.imSession.serve();

	return

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	api.Bind(e)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}
