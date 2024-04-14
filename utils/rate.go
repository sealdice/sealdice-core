package utils

import (
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

func ParseRate(s string) (rate.Limit, error) {
	// 为了防止奇怪的用户输入，还是先固定这种格式吧
	if strings.HasPrefix(s, "@every ") {
		durStr := strings.TrimPrefix(s, "@every ")
		dur, err := time.ParseDuration(durStr)
		if err != nil {
			return 0, err
		}
		return rate.Every(dur), nil
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return rate.Limit(n), nil
}
