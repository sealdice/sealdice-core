package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"os"
	"os/signal"
	"sealdice-core/api"
	"sealdice-core/core"
	"sealdice-core/dice"
	"sealdice-core/model"
	"syscall"
	"time"
)

/**
二进制目录结构:
data/configs
data/extensions
data/logs

extensions/
*/

func main() {
	core.LoggerInit()
	model.DBInit()
	cleanUp := func() {
		fmt.Println("程序即将推出，进行清理……")
		model.GetDB().Close()
	}
	defer cleanUp()

	go (func() {
		// 创建核心并提供服务
		myDice := &dice.Dice{}
		myDice.Init()

		for {
			// 骰子开始连接
			fmt.Println("开始连接 onebot 服务")
			ret := myDice.ImSession.Serve()

			if ret == 0 {
				break
			}

			fmt.Println("onebot 连接中断，将在15秒后重新连接")
			time.Sleep(time.Duration(15 * time.Second))
		}
	})()

	go (func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-interrupt:
			time.Sleep(time.Duration(5 * time.Second))
			fmt.Println("仍未关闭，强制退出")
			cleanUp()
			os.Exit(0)
		}
	})()

	fmt.Println("即将启动webui")
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
