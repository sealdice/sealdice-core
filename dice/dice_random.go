package dice

import (
	"errors"

	ds "github.com/sealdice/dicescript"

	randcore "sealdice-core/utils/random"
)

func (d *Dice) getDiceRandomMode() DiceRandomMode {
	return randcore.NormalizeMode(d.Config.DiceRandomMode)
}

// migrateDiceRandomGMConfig 保持旧 gm/hybrid 配置的原有行为：升级后自动启用国密源。
// hasGMField 用于识别配置中是否真的写入了新字段，避免旧版程序回写时丢弃字段却保留新版本号。
func (d *Dice) migrateDiceRandomGMConfig(hasGMField bool) bool {
	if hasGMField {
		if d.Config.VersionCode >= ConfigVersionCode {
			return false
		}
		d.Config.VersionCode = ConfigVersionCode
		return true
	}

	mode := d.getDiceRandomMode()
	if mode == DiceRandomModeGM || mode == DiceRandomModeHybrid {
		d.Config.DiceRandomGMEnabled = true
		if d.Logger != nil {
			d.Logger.Warnf(
				"[随机源] 检测到旧配置使用 %s 随机源，已自动启用国密随机源以保持原有行为。国密随机源需要约 32MB 常驻内存，用于采集系统熵并保证 SM3 随机数的随机性和不可预测性；这部分内存启用后会一直占用，只有重启海豹才能释放。如不需要，请执行 .randalgo set gm off 后重启海豹。",
				randcore.ModeSpecFor(mode).Label,
			)
		}
	}
	d.Config.VersionCode = ConfigVersionCode
	return true
}

func (d *Dice) EnableDiceRandomGM() error {
	if err := globalRandSource.EnableGM(d.Logger); err != nil {
		return err
	}
	d.Config.DiceRandomGMEnabled = true
	return nil
}

func (d *Dice) DisableDiceRandomGM() error {
	if err := globalRandSource.DisableGM(d.Logger); err != nil {
		return err
	}
	d.Config.DiceRandomGMEnabled = false
	if d.getDiceRandomMode() == DiceRandomModeGM {
		d.Config.DiceRandomMode = string(DiceRandomModePCG)
		return d.ActivateDiceRandomMode()
	}
	return nil
}

func (d *Dice) ActivateDiceRandomMode() error {
	if d == nil {
		return errors.New("dice is nil")
	}

	configuredMode := d.getDiceRandomMode()
	effectiveMode, initErr := globalRandSource.SetActive(configuredMode)
	if effectiveMode != configuredMode && d.Logger != nil {
		d.Logger.Warnf(
			"[随机源] 配置模式 %s 不可用，当前运行时已回退到 %s。错误: %v",
			configuredMode,
			effectiveMode,
			initErr,
		)
	}
	globalRandSource.LogActiveMode(d.Logger)
	if effectiveMode != configuredMode {
		return initErr
	}
	return nil
}

func (d *Dice) Roll(points int) int {
	if points <= 0 {
		return 0
	}
	val := ds.Roll(globalRandSource, ds.IntType(points), 0)
	return int(val)
}

func (d *Dice) Roll64(points int64) int64 {
	return DiceRoll64x(globalRandSource, points)
}

func (d *Dice) RandIntn(n int) int {
	return randIntnFromSource(globalRandSource, n)
}

func (d *Dice) Shuffle(n int, swap func(i, j int)) {
	shuffleWithSource(globalRandSource, n, swap)
}
