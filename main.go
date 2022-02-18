package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"sealdice-core/api"
	"sealdice-core/core"
)


func main() {
	core.LoggerInit()
	//model.DBInit()
	//defer model.GetDB().Close()
	//
	//exp := "(10d1)d(3+5-7)"
	////exp := "1234^4+15*6-17+6d12+3d15k2-d12"
	//dep := &DiceRollParser{ Buffer: exp }
	//_ = dep.Init()
	//
	//dep.RollExpression.Init(1000)
	//if err := dep.Parse(); err != nil {
	//	panic(err)
	//}
	//
	//dep.Execute()
	//fmt.Println(dep.Evaluate());

	//return;
	//dep.RollExpression.Execute()

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
	configInit()

	dice := &Dice{};
	dice.init();

	//1+2*3+d1+4+1d5+
	//a, d, err := dice.exprEval("0 + 100 + 10 + 2d5 + 2*3*1d5 + 1d4 + 7 + 力量", nil)

	//a, d, err := dice.exprEval("1d30+1", nil)
	//a, d, err := dice.exprEval("`ROLL:{ 1d20 } LIFE:{ 3d20 } test: { 4 + 5 + 力量}`", nil)
	//a, d, err := dice.exprEval("`ROLL:{ 1d20 } LIFE:{ 4 }`", nil)
	//a, d, err := dice.exprEval("+", nil)
	a, d, err := dice.exprEval("测试", nil)
	if err == nil {
		fmt.Println("DDD" + "#{a}", a.typeId, a.value, d, err)
	} else {
		fmt.Println("DDD2", err)
	}

	dice.ImSession.serve();

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
