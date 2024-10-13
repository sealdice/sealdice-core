package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kardianos/service"

	log "sealdice-core/utils/kratos"
)

type program struct{}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() {
	// Do work here
	main()
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

func serviceInstall(isInstall bool, serviceName string, user string) {
	cwd, _ := os.Getwd()
	wd, _ := filepath.Abs(cwd)

	if serviceName == "" {
		serviceName = "sealdice"
	}
	svcConfig := &service.Config{
		Name:             serviceName,
		DisplayName:      "SealDice Service",
		Description:      "SealDice: A TRPG Dice Bot Service.",
		WorkingDirectory: wd,
	}

	if user != "" {
		svcConfig.UserName = user
	}

	prg := &program{}
	log.Info("正在试图访问系统服务 ...")
	s, err := service.New(prg, svcConfig)

	if isInstall {
		log.Info("正在安装系统服务，安装完成后，SealDice将自动随系统启动")
		if err != nil {
			fmt.Printf("安装失败: %s\n", err.Error())
		}
		_, err = s.Logger(nil)
		if err != nil {
			fmt.Printf("安装失败: %s\n", err.Error())
			log.Info(err)
		}
		err = s.Install()
		if err != nil {
			fmt.Printf("安装失败: %s\n", err.Error())
			return
		}

		log.Info("安装完成，正在启动……")
		_ = s.Start()
	} else {
		log.Info("正在卸载系统服务……")
		_ = s.Stop()
		_ = s.Uninstall()
		log.Info("系统服务已删除")
	}
}
