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

	case "official":
		conn := ep.Adapter.(*PlatformAdapterOfficialQQ)
		serverOfficialQQ(d, ep, conn)
	case "onebot":
		fallthrough
	default: // onebot 作为默认情况
		conn := ep.Adapter.(*PlatformAdapterGocq)
		serverGocq(d, ep, conn)
	}
}

func serverGocq(d *Dice, ep *EndPointInfo, conn *PlatformAdapterGocq) {
	conn.runtimeMu.Lock()
	if conn.runtimeStopping.Load() || conn.diceServing {
		conn.runtimeMu.Unlock()
		return
	}
	conn.diceServing = true
	ep.Enable = true
	ep.State = 2 // 连接中
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	conn.runtimeMu.Unlock()

	checkQuit := func() bool {
		if conn.runtimeStopping.Load() {
			return true
		}
		if conn.GoCqhttpState == StateCodeInLoginDeviceLock {
			d.Logger.Infof("检测到设备锁流程，暂时不再连接")
			ep.State = 0
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
			return true
		}
		conn.runtimeMu.Lock()
		serving := conn.diceServing
		conn.runtimeMu.Unlock()
		if !serving {
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
	for !checkQuit() {
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
			if !waitRuntimeDelay(diceRuntimeContext(d), 15*time.Second) {
				break
			}
			continue
		}

		conn.reconnectTimes++
		if conn.reconnectTimes > 5 {
			d.Logger.Infof("onebot 连接重试次数过多，先行中断: <%s>(%s)", ep.Nickname, ep.UserID)
			ep.State = 0
			conn.GoCqhttpState = StateCodeLoginFailed
			break
		}

		if !waitRuntimeDelay(diceRuntimeContext(d), 15*time.Second) {
			break
		}
	}

	conn.runtimeMu.Lock()
	conn.diceServing = false
	conn.runtimeMu.Unlock()
}

func serverWalleQ(d *Dice, ep *EndPointInfo, conn *PlatformAdapterWalleQ) {
	conn.runtimeMu.Lock()
	if conn.runtimeStopping.Load() || conn.DiceServing {
		conn.runtimeMu.Unlock()
		return
	}
	conn.DiceServing = true
	ep.Enable = true
	ep.State = 2 // 连接中
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	conn.runtimeMu.Unlock()

	checkQuit := func() bool {
		if conn.runtimeStopping.Load() {
			return true
		}
		if conn.WalleQState == StateCodeInLoginDeviceLock {
			d.Logger.Infof("检测到设备锁流程，暂时不再连接")
			ep.State = 0
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
			return true
		} // 暂时去掉设备锁检查
		conn.runtimeMu.Lock()
		serving := conn.DiceServing
		conn.runtimeMu.Unlock()
		if !serving {
			// 退出连接
			d.Logger.Infof("检测到连接关闭，不再进行此onebot服务的重连: <%s>(%s)", ep.Nickname, ep.UserID)
			return true
		}
		return false
	}

	waitTimes := 0
	for !checkQuit() {
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
			conn.runtimeMu.Lock()
			conn.DiceServing = false
			conn.runtimeMu.Unlock()
			break
		}

		if !waitRuntimeDelay(diceRuntimeContext(d), 15*time.Second) {
			break
		}
	}
	conn.runtimeMu.Lock()
	conn.DiceServing = false
	conn.runtimeMu.Unlock()
}

func serverRed(d *Dice, ep *EndPointInfo, conn *PlatformAdapterRed) {
	conn.runtimeMu.Lock()
	if conn.runtimeStopping.Load() || conn.DiceServing {
		conn.runtimeMu.Unlock()
		return
	}
	conn.DiceServing = true
	ep.Enable = true
	ep.State = 2 // 连接中
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	conn.runtimeMu.Unlock()
	waitTimes := 0

	for !conn.runtimeStopping.Load() {
		// 骰子开始连接
		d.Logger.Infof("开始连接 red 服务，帐号 <%s>(%s)，重试计数[%d/%d]", ep.Nickname, ep.UserID, waitTimes, 5)
		ret := ep.Adapter.Serve()

		if ret == 0 {
			break
		}
		if conn.runtimeStopping.Load() {
			break
		}

		waitTimes += 1
		if waitTimes > 5 {
			d.Logger.Infof("red 连接重试次数过多，先行中断: <%s>(%s)", ep.Nickname, ep.UserID)
			break
		}

		if !waitRuntimeDelay(diceRuntimeContext(d), 15*time.Second) {
			break
		}
	}
	conn.runtimeMu.Lock()
	conn.DiceServing = false
	conn.runtimeMu.Unlock()
}

func serverSatori(d *Dice, ep *EndPointInfo, conn *PlatformAdapterSatori) {
	if diceRuntimeContext(d).Err() != nil {
		return
	}
	if conn.DiceServing {
		return
	}
	conn.DiceServing = true

	ep.Enable = true
	ep.State = 2 // 连接中
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	waitTimes := 0

	for ep.State == 2 && diceRuntimeContext(d).Err() == nil {
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

		if !waitRuntimeDelay(diceRuntimeContext(d), 15*time.Second) {
			break
		}
	}
	conn.DiceServing = false
}

func serverOfficialQQ(d *Dice, ep *EndPointInfo, conn *PlatformAdapterOfficialQQ) {
	conn.runtimeMu.Lock()
	// Ctx 非空表示会话已经建立或正在运行，不能重复启动并覆盖端点状态。
	if conn.runtimeStopping.Load() || conn.Ctx != nil {
		conn.runtimeMu.Unlock()
		return
	}
	if conn.DiceServing {
		conn.runtimeMu.Unlock()
		return
	}
	conn.DiceServing = true

	ep.Enable = true
	ep.State = 2 // 连接中
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	conn.runtimeMu.Unlock()
	waitTimes := 0

	for !conn.runtimeStopping.Load() {
		// 骰子开始连接
		d.Logger.Infof("开始连接 official qq，帐号 <%s>(%s)，重试计数[%d/%d]", ep.Nickname, ep.UserID, waitTimes, 5)
		ret := ep.Adapter.Serve()

		if ret == 0 {
			break
		}
		if conn.runtimeStopping.Load() {
			break
		}

		waitTimes += 1
		if waitTimes > 5 {
			d.Logger.Infof("official qq 连接重试次数过多，先行中断: <%s>(%s)", ep.Nickname, ep.UserID)
			break
		}

		if !waitRuntimeDelay(diceRuntimeContext(d), 15*time.Second) {
			break
		}
	}
	conn.runtimeMu.Lock()
	conn.DiceServing = false
	conn.runtimeMu.Unlock()
}
