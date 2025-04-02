package senders

// CalculateMessagePartsLength calculates and returns the length of the
// description and events string in order to fit the max chars limit.
func CalculateMessagePartsLength(maxChars, descLen, eventsLen int) (descNewLen int, eventsNewLen int) {
	if descLen+eventsLen <= maxChars {
		return descLen, eventsLen
	}

	halfOfMaxChars := maxChars / partsCountForMessageWithDescAndEvents

	if descLen > halfOfMaxChars && eventsLen <= halfOfMaxChars {
		return maxChars - eventsLen - 10, eventsLen
	}

	if eventsLen > halfOfMaxChars && descLen <= halfOfMaxChars {
		return descLen, maxChars - descLen
	}

	return halfOfMaxChars - 10, halfOfMaxChars
}

const (
	// partsCountForMessageWithDescAndEvents is used then you need to split given maxChars fairly by half
	// between description and events.
	partsCountForMessageWithDescAndEvents = 2
	// partsCountForMessageWithTagsDescAndEvents is used then you need to split given maxChars fairly by three parts
	// between tags, description and events.
	partsCountForMessageWithTagsDescAndEvents = 3
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

	fairMaxLen := maxChars / partsCountForMessageWithTagsDescAndEvents

	switch {
	case tagsLen > fairMaxLen && descLen <= fairMaxLen && eventsLen <= fairMaxLen:
		// give free space to tags
		tagsNewLen = maxChars - descLen - eventsLen

		return min(tagsNewLen, tagsLen), descLen, eventsLen
	case tagsLen <= fairMaxLen && descLen > fairMaxLen && eventsLen <= fairMaxLen:
		// give free space to description
		descNewLen = maxChars - tagsLen - eventsLen

		return tagsLen, min(descNewLen, descLen), eventsLen
	case tagsLen <= fairMaxLen && descLen <= fairMaxLen && eventsLen > fairMaxLen:
		// give free space to events
		eventsNewLen = maxChars - tagsLen - descLen

		return tagsLen, descLen, min(eventsNewLen, eventsLen)
	case tagsLen > fairMaxLen && descLen > fairMaxLen && eventsLen <= fairMaxLen:
		// description is more important than tags
		tagsNewLen = fairMaxLen
		descNewLen = maxChars - tagsNewLen - eventsLen

		return tagsNewLen, min(descNewLen, descLen), eventsLen
	case tagsLen > fairMaxLen && descLen <= fairMaxLen && eventsLen > fairMaxLen:
		// events are more important than tags
		tagsNewLen = fairMaxLen
		eventsNewLen = maxChars - tagsNewLen - descLen

		return tagsNewLen, descLen, min(eventsNewLen, eventsLen)
	case tagsLen <= fairMaxLen && descLen > fairMaxLen && eventsLen > fairMaxLen:
		// split free space from tags fairly between description and events
		spaceFromTags := fairMaxLen - tagsLen
		halfOfSpaceFromTags := spaceFromTags / partsCountForMessageWithDescAndEvents

		descNewLen = fairMaxLen + halfOfSpaceFromTags
		eventsNewLen = fairMaxLen + halfOfSpaceFromTags

		return tagsLen, min(descNewLen, descLen), min(eventsNewLen, eventsLen)
	default:
		// all 3 blocks have length greater than maxChars/3, so split space fairly
		return fairMaxLen, fairMaxLen, fairMaxLen
	}
}
