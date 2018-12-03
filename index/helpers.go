package index

import (
	"regexp"
	"strings"
	"time"
)

const (
	indexWaitTimeout = time.Second * 3
)

func (index *Index) checkIfIndexIsReady() bool {
	if index.IsReady() {
		return true
	}
	timeout := time.After(indexWaitTimeout)
	ticker := time.NewTicker(time.Second * 1)

	for {
		select {
		case <-ticker.C:
			if index.IsReady() {
				return true
			}
		case <-timeout:
			return index.IsReady()
		}
	}
}

func splitStringToTerms(searchString string) (searchTerms []string) {
	searchString = escapeString(searchString)

	return strings.Fields(searchString)
}

func escapeString(original string) (escaped string) {
	return regexp.MustCompile(`[|+\-=&<>!(){}\[\]^"'~*?\\/.,:;_@]`).ReplaceAllString(original, " ")
}
