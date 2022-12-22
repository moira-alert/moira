package senders

// CalculateMessagePartsLength calculates and returns the length of the
// description and events string in order to fit the max chars limit
func CalculateMessagePartsLength(maxChars, descLen, eventsLen int) (descNewLen int, eventsNewLen int) {
	if descLen+eventsLen <= maxChars {
		return descLen, eventsLen
	}
	if descLen > maxChars/2 && eventsLen <= maxChars/2 {
		return maxChars - eventsLen - 10, eventsLen
	}
	if eventsLen > maxChars/2 && descLen <= maxChars/2 {
		return descLen, maxChars - descLen
	}
	return maxChars/2 - 10, maxChars / 2
}
