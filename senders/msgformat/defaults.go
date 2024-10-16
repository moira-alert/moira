package msgformat

import "unicode/utf8"

// DefaultDescriptionCutter cuts description, so len(newDesc) <= maxSize. Ensure that len(desc) >= maxSize and
// maxSize >= len("...\n").
func DefaultDescriptionCutter(desc string, maxSize int) string {
	suffix := "...\n"
	return desc[:maxSize-len(suffix)] + suffix
}

var bracketsLen = utf8.RuneCountInString("[]")

// DefaultTagsLimiter cuts and formats tags to fit maxSize. There will be no tag parts, for example:
//
// if we have
//
//	tags = []string{"tag1", "tag2}
//	maxSize = 8
//
// so call DefaultTagsLimiter(tags, maxSize) will return " [tag1]".
func DefaultTagsLimiter(tags []string, maxSize int) string {
	tagsStr := " "
	lenTagsStr := utf8.RuneCountInString(tagsStr)

	for i := range tags {
		lenTag := utf8.RuneCountInString(tags[i]) + bracketsLen

		if lenTagsStr+lenTag > maxSize {
			break
		}

		tagsStr += "[" + tags[i] + "]"
		lenTagsStr += lenTag
	}

	if tagsStr == " " {
		return ""
	}

	return tagsStr
}
