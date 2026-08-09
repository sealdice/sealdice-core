//go:build windows || darwin

package main

import (
	"fmt"

	"sealdice-core/dice"
)

const defaultTrayTooltip = "海豹TRPG骰点核心"

func formatTrayTooltip(dm *dice.DiceManager, version, port string) string {
	prefix := defaultTrayTooltip
	if dm != nil {
		if text := dice.NormalizeTrayTooltipPrefix(dm.GetTrayTooltip()); text != "" {
			prefix = text
		}
	}
	return fmt.Sprintf("%s %s #%s", prefix, version, port)
}
