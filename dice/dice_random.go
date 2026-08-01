package dice

import (
	"errors"

	ds "github.com/sealdice/dicescript"

	randcore "sealdice-core/utils/random"
)

func (d *Dice) getDiceRandomMode() DiceRandomMode {
	return randcore.NormalizeMode(d.Config.DiceRandomMode)
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
