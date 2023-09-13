package dice

import (
	"math"
	"math/rand"
)

// Roll 获得均匀分布的随机整数
func Roll(dicePoints int) int {
	if dicePoints <= 0 {
		return 0
	}
	return int(Roll64(int64(dicePoints)))
}

// Roll64 获得均匀分布的随机整数
func Roll64(dicePoints int64) int64 {
	if dicePoints == 0 {
		return 0
	}
	val := rand.Int63()%dicePoints + 1
	return val
}

// RollGauss 获得符合正态分布的随机整数，使用 Box-Muller 变换
func RollGauss(dicePoints int) int {
	if dicePoints == 0 {
		return 0
	}
	return int(RollGauss64(int64(dicePoints)))
}

// RollGauss64 获得符合正态分布的随机整数，使用 Box-Muller 变换
func RollGauss64(dicePoints int64) int64 {
	if dicePoints == 0 {
		return 0
	}
	x := rand.Float64()
	y := rand.Float64()
	z1 := math.Sqrt(-2*math.Log(x)) * math.Cos(2*math.Pi*y)
	z2 := math.Sqrt(-2*math.Log(x)) * math.Sin(2*math.Pi*y)

	// 使得 95% 的值落在区间
	mu := float64(dicePoints) / 2    // (max-min)/2
	sigma := float64(dicePoints) / 4 //（max-min/4

	r := int64(math.Round(z1*sigma + mu))
	// 保证范围，第一个不在用第二个，第二个不在就再来一次
	if r <= 0 || r > dicePoints {
		r := int64(math.Round(z2*sigma + mu))
		if r <= 0 || r > dicePoints {
			return RollGauss64(dicePoints)
		} else {
			return r
		}
	} else {
		return r
	}
}
