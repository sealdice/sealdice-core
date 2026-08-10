package dice

import randcore "sealdice-core/utils/random"

type DiceRandomMode = randcore.Mode

const (
	DiceRandomModePCG    = randcore.ModePCG
	DiceRandomModeGM     = randcore.ModeGM
	DiceRandomModeNIST   = randcore.ModeNIST
	DiceRandomModeCRNG   = randcore.ModeCRNG
	DiceRandomModeHybrid = randcore.ModeHybrid
)

var supportedDiceRandomModes = randcore.SupportedModes()

func parseDiceRandomModeStrict(raw string) (DiceRandomMode, bool) {
	return randcore.ParseModeStrict(raw)
}

func NormalizeDiceRandomMode(raw string) DiceRandomMode {
	return randcore.NormalizeMode(raw)
}
