package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-graphite/carbonapi/date"
)

func validateFromStr(fromStr string) (string, error) {
	if fromStr != "-inf" {
		from := date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
		if from == 0 {
			return "", fmt.Errorf("can not parse from: %s", fromStr)
		}
		fromStr = strconv.FormatInt(from, 10)
	}

	return fromStr, nil
}

func validateToStr(toStr string) (string, error) {
	if toStr != "+inf" {
		to := date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
		if to == 0 {
			return "", fmt.Errorf("can not parse to: %v", to)
		}
		toStr = strconv.FormatInt(to, 10)
	}

	return toStr, nil
}
