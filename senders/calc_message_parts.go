package senders

import (
	"math"
)

// percent for event
const percentForEvent = 0.3

// CalculateMessagePartsLength calculates and returns the length of the
// description and events string in order to fit the max chars limit
func CalculateMessagePartsLength(maxChars, descLen, eventsLen int) (descNewLen int, eventsNewLen int) {
	charsForEvents := int(math.Round(float64(maxChars) * percentForEvent))

	switch {
	case descLen+eventsLen <= maxChars:
		return descLen, eventsLen
	case eventsLen <= charsForEvents:
		return maxChars - eventsLen, eventsLen
	default:
		return maxChars - charsForEvents, charsForEvents
	}
}
