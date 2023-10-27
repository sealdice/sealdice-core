package dice

import "time"

func ServeQQ(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	// 换成 gocq 是不是更好 and 函数名都叫 ServeQQ 了……
	if ep.Platform == "QQ" && (ep.ProtocolType == "onebot" || ep.ProtocolType == "") {
		conn := ep.Adapter.(*PlatformAdapterGocq)

		if !conn.DiceServing {
			conn.DiceServing = true
		} else {
			return
		}

		ep.Enable = true
		ep.State = 2 // 连接中
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)

		checkQuit := func() bool {
			if conn.GoCqhttpState == StateCodeInLoginDeviceLock {
				d.Logger.Infof("检测到设备锁流程，暂时不再连接")
				ep.State = 0
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
				return true
			}
			if !conn.DiceServing {
				// 退出连接
				d.Logger.Infof("检测到连接关闭，不再进行此onebot服务的重连: <%s>(%s)", ep.Nickname, ep.UserID)
				return true
			}
			return false
		}

		waitTimes := 0
		for {
			if checkQuit() {
				break
			}

			// 骰子开始连接
			d.Logger.Infof("开始连接 onebot 服务，帐号 <%s>(%s)，重试计数[%d/%d]", ep.Nickname, ep.UserID, waitTimes, 5)
			ret := ep.Adapter.Serve()

			if ret == 0 {
				break
			}

			if checkQuit() {
				break
			}

			waitTimes++
			if waitTimes > 5 {
				d.Logger.Infof("onebot 连接重试次数过多，先行中断: <%s>(%s)", ep.Nickname, ep.UserID)
				conn.DiceServing = false
				break
			}

			time.Sleep(15 * time.Second)
		}
	}
	// TODO 重复了，写个函数
	if ep.Platform == "QQ" && ep.ProtocolType == "walle-q" {
		conn := ep.Adapter.(*PlatformAdapterWalleQ)

		if !conn.DiceServing {
			conn.DiceServing = true
		} else {
			return
		}

		ep.Enable = true
		ep.State = 2 // 连接中
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)

		checkQuit := func() bool {
			if conn.WalleQState == StateCodeInLoginDeviceLock {
				d.Logger.Infof("检测到设备锁流程，暂时不再连接")
				ep.State = 0
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
				return true
			} // 暂时去掉设备锁检查
			if !conn.DiceServing {
				// 退出连接
				d.Logger.Infof("检测到连接关闭，不再进行此onebot服务的重连: <%s>(%s)", ep.Nickname, ep.UserID)
				return true
			}
			return false
		}

		waitTimes := 0
		for {
			if checkQuit() {
				break
			}

			// 骰子开始连接
			d.Logger.Infof("开始连接 onebot 服务，帐号 <%s>(%s)，重试计数[%d/%d]", ep.Nickname, ep.UserID, waitTimes, 5)
			ret := ep.Adapter.Serve()

			if ret == 0 {
				break
			}

			if checkQuit() {
				break
			}

			waitTimes++
			if waitTimes > 5 {
				d.Logger.Infof("onebot 连接重试次数过多，先行中断: <%s>(%s)", ep.Nickname, ep.UserID)
				conn.DiceServing = false
				break
			}

			time.Sleep(15 * time.Second)
		}
	}

	if ep.Platform == "QQ" && ep.ProtocolType == "red" {
		conn := ep.Adapter.(*PlatformAdapterRed)

		if !conn.DiceServing {
			conn.DiceServing = true
		} else {
			return
		}

		ep.Enable = true
		ep.State = 2 // 连接中
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		waitTimes := 0

		for {
			// 骰子开始连接
			d.Logger.Infof("开始连接 red 服务，帐号 <%s>(%s)，重试计数[%d/%d]", ep.Nickname, ep.UserID, waitTimes, 5)
			ret := ep.Adapter.Serve()

			if ret == 0 {
				break
			}

			waitTimes += 1
			if waitTimes > 5 {
				d.Logger.Infof("red 连接重试次数过多，先行中断: <%s>(%s)", ep.Nickname, ep.UserID)
				conn.DiceServing = false
				break
			}

			time.Sleep(15 * time.Second)
		}
	}
}
