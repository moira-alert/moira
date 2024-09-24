package senders

// CalculateMessagePartsLength calculates and returns the length of the
// description and events string in order to fit the max chars limit.
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

const (
	// messagePartsCount is the number of parts in alert, which will be recalculated.
	messagePartsCount = 3
)

// CalculateMessagePartsBetweenTagsDescEvents calculates and returns the length of tags, description and events string
// in order to fit the max chars limit.
func CalculateMessagePartsBetweenTagsDescEvents(maxChars, tagsLen, descLen, eventsLen int) (tagsNewLen int, descNewLen int, eventsNewLen int) { // nolint
	if maxChars <= 0 {
		return 0, 0, 0
	}

	if tagsLen+descLen+eventsLen <= maxChars {
		return tagsLen, descLen, eventsLen
	}

	fairMaxLen := maxChars / messagePartsCount

	switch {
	case firstIsGreaterThanGivenLenAndOthersLessOrEqual(fairMaxLen, tagsLen, descLen, eventsLen):
		// give free space to tags
		tagsNewLen = maxChars - descLen - eventsLen

		return min(tagsNewLen, tagsLen), descLen, eventsLen
	case firstIsGreaterThanGivenLenAndOthersLessOrEqual(fairMaxLen, descLen, tagsLen, eventsLen):
		// give free space to description
		descNewLen = maxChars - tagsLen - eventsLen

		return tagsLen, min(descNewLen, descLen), eventsLen
	case firstIsGreaterThanGivenLenAndOthersLessOrEqual(fairMaxLen, eventsLen, tagsLen, descLen):
		// give free space to events
		eventsNewLen = maxChars - tagsLen - descLen

		return tagsLen, descLen, min(eventsNewLen, eventsLen)
	case firstAndSecondIsGreaterThanGivenLenAndOtherLessOrEqual(fairMaxLen, tagsLen, descLen, eventsLen):
		// description is more important than tags
		tagsNewLen = fairMaxLen
		descNewLen = maxChars - tagsNewLen - eventsLen

		return tagsNewLen, min(descNewLen, descLen), eventsLen
	case firstAndSecondIsGreaterThanGivenLenAndOtherLessOrEqual(fairMaxLen, tagsLen, eventsLen, descLen):
		// events are more important than tags
		tagsNewLen = fairMaxLen
		eventsNewLen = maxChars - tagsNewLen - descLen

		return tagsNewLen, descLen, min(eventsNewLen, eventsLen)
	case firstAndSecondIsGreaterThanGivenLenAndOtherLessOrEqual(fairMaxLen, descLen, eventsLen, tagsLen):
		// split free space from tags fairly between description and events
		spaceFromTags := fairMaxLen - tagsLen
		descNewLen = fairMaxLen + spaceFromTags/2
		eventsNewLen = fairMaxLen + spaceFromTags/2

		return tagsLen, min(descNewLen, descLen), min(eventsNewLen, eventsLen)
	default:
		// all 3 blocks have length greater than maxChars/3, so split space fairly
		return fairMaxLen, fairMaxLen, fairMaxLen
	}
}

func firstIsGreaterThanGivenLenAndOthersLessOrEqual(givenLen, first, second, third int) bool {
	return first > givenLen && second <= givenLen && third <= givenLen
}

func firstAndSecondIsGreaterThanGivenLenAndOtherLessOrEqual(givenLen, first, second, third int) bool {
	return first > givenLen && second > givenLen && third <= givenLen
}
