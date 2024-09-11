package msgformat

import "unicode/utf8"

// DefaultDescriptionCutter cuts description, so len(newDesc) <= maxSize. Ensure that len(desc) >= maxSize and
// maxSize >= len("...\n").
func DefaultDescriptionCutter(desc string, maxSize int) string {
	suffix := "...\n"
	return desc[:maxSize-len(suffix)] + suffix
}

func DefaultTagsLimiter(tags []string, maxSize int) string {
	tagsStr := " "
	lenTagsStr := utf8.RuneCountInString(tagsStr)

	for i := range tags {
		lenTag := utf8.RuneCountInString(tags[i]) + 2

		if lenTagsStr+lenTag > maxSize {
			break
		}

		tagsStr += "[" + tags[i] + "]"
		lenTagsStr += lenTag

		if lenTagsStr == maxSize {
			break
		}
	}

	if tagsStr == " " {
		return ""
	}

	return tagsStr
}
