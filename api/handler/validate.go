package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-graphite/carbonapi/date"
)

// DateRangeValidator for query parameters from and to.
type DateRangeValidator struct {
	AllowInf bool
}

// ValidateDateRangeStrings validates and converts both from and to query params.
// Returns converted from and to, or error if there was any.
// If AllowInf is true "-inf" is allowed for from, "+inf" for to.
func (d DateRangeValidator) ValidateDateRangeStrings(fromStr, toStr string) (string, string, error) {
	fromStr, err := d.validateFromStr(fromStr)
	if err != nil {
		return "", "", err
	}

	toStr, err = d.validateToStr(toStr)
	if err != nil {
		return "", "", err
	}

	return fromStr, toStr, nil
}

// validateFromStr by trying to parse date with carbonapi/date package. Also converts to proper format.
// If AllowInf is true, then "-inf" is also allowed.
func (d DateRangeValidator) validateFromStr(fromStr string) (string, error) {
	if d.AllowInf && fromStr == "-inf" {
		return fromStr, nil
	}

	from := date.DateParamToEpoch(fromStr, "UTC", 0, time.UTC)
	if from == 0 {
		return "", fmt.Errorf("can not parse from: %s", fromStr)
	}

	return strconv.FormatInt(from, 10), nil
}

// validateToStr by trying to parse date with carbonapi/date package. Also converts to proper format.
// If AllowInf is true, then "+inf" is also allowed.
func (d DateRangeValidator) validateToStr(toStr string) (string, error) {
	if d.AllowInf && toStr == "+inf" {
		return toStr, nil
	}

	to := date.DateParamToEpoch(toStr, "UTC", 0, time.UTC)
	if to == 0 {
		return "", fmt.Errorf("can not parse to: %v", to)
	}

	return strconv.FormatInt(to, 10), nil
}
