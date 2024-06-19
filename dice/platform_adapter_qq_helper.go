package dice

import (
	"time"
)

func ServeQQ(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform != "QQ" {
		return
	}

	switch ep.ProtocolType {
	case "walle-q":
		conn := ep.Adapter.(*PlatformAdapterWalleQ)
		serverWalleQ(d, ep, conn)

	case "red":
		conn := ep.Adapter.(*PlatformAdapterRed)
		serverRed(d, ep, conn)

	case "satori":
		conn := ep.Adapter.(*PlatformAdapterSatori)
		serverSatori(d, ep, conn)

	case "onebot":
		fallthrough
	default: // onebot 作为默认情况
		conn := ep.Adapter.(*PlatformAdapterGocq)
		serverGocq(d, ep, conn)
	}
}

func serverGocq(d *Dice, ep *EndPointInfo, conn *PlatformAdapterGocq) {
	if conn.diceServing {
		return
	}
	conn.diceServing = true

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
		if !conn.diceServing {
			// 退出连接
			d.Logger.Infof("检测到连接关闭，不再进行此onebot服务的重连: <%s>(%s)", ep.Nickname, ep.UserID)
			return true
		}
		if conn.GoCqhttpState == StateCodeLoginFailed {
			d.Logger.Infof("检测到登录失败，不再进行此onebot服务的重连: <%s>(%s)", ep.Nickname, ep.UserID)
			return true
		}
		if !ep.Enable {
			d.Logger.Infof("检测到账号被禁用，不再进行此onebot服务的重连: <%s>(%s)", ep.Nickname, ep.UserID)
			return true
		}
		return false
	}

	conn.reconnectTimes = 0
	for {
		if checkQuit() {
			break
		}

		// 骰子开始连接
		d.Logger.Infof("开始连接 onebot 服务，帐号 <%s>(%s)，重试计数[%d/%d]", ep.Nickname, ep.UserID, conn.reconnectTimes, 5)
		ret := ep.Adapter.Serve()

		if ret == 0 {
			break
		}

		if checkQuit() {
			break
		}

		if conn.GoCqhttpState == StateCodeInLogin || conn.GoCqhttpState == StateCodeInLoginQrCode {
			time.Sleep(15 * time.Second)
			continue
		}

		conn.reconnectTimes++
		if conn.reconnectTimes > 5 {
			d.Logger.Infof("onebot 连接重试次数过多，先行中断: <%s>(%s)", ep.Nickname, ep.UserID)
			ep.State = 0
			conn.GoCqhttpState = StateCodeLoginFailed
			break
		}

		time.Sleep(15 * time.Second)
	}

	conn.diceServing = false
}

func serverWalleQ(d *Dice, ep *EndPointInfo, conn *PlatformAdapterWalleQ) {
	if conn.DiceServing {
		return
	}
	conn.DiceServing = true
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

func serverRed(d *Dice, ep *EndPointInfo, conn *PlatformAdapterRed) {
	if conn.DiceServing {
		return
	}
	conn.DiceServing = true

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

func serverSatori(d *Dice, ep *EndPointInfo, conn *PlatformAdapterSatori) {
	if conn.DiceServing {
		return
	}
	conn.DiceServing = true

	ep.Enable = true
	ep.State = 2 // 连接中
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	waitTimes := 0

	for {
		if ep.State != 2 {
			break
		}
		// 骰子开始连接
		d.Logger.Infof("开始连接 satori 服务，帐号 <%s>(%s)，重试计数[%d/%d]", ep.Nickname, ep.UserID, waitTimes, 5)
		ret := ep.Adapter.Serve()

		if ret == 0 {
			break
		}

		waitTimes += 1
		if waitTimes > 5 {
			d.Logger.Infof("satori 连接重试次数过多，先行中断: <%s>(%s)", ep.Nickname, ep.UserID)
			conn.DiceServing = false
			break
		}

		time.Sleep(15 * time.Second)
	}
}
