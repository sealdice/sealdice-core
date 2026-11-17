package dice

import (
	"strconv"
	"strings"
)

type qqBotUINRange struct {
	min int64
	max int64
}

var qqBotUINRanges = [...]qqBotUINRange{
	{min: 3328144510, max: 3328144510},
	{min: 2854196301, max: 2854216399},
	{min: 66600000, max: 66600000},
	{min: 3889000000, max: 3889999999},
	{min: 4010000000, max: 4019999999},
}

func isQQBotUIN(uin int64) bool {
	for _, uinRange := range qqBotUINRanges {
		if uin >= uinRange.min && uin <= uinRange.max {
			return true
		}
	}
	return false
}

func isQQBotUserID(userID string) bool {
	rawUIN, ok := strings.CutPrefix(userID, "QQ:")
	if !ok || rawUIN == "" {
		return false
	}
	for _, digit := range rawUIN {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	uin, err := strconv.ParseInt(rawUIN, 10, 64)
	return err == nil && isQQBotUIN(uin)
}
