package index

import (
	"strings"
	"time"
)

const (
	symbolsToEscape  = `|+-=&<>!(){}[]^"'~*?\/.,:;_-@`
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
	escaped = original
	for _, symbol := range symbolsToEscape {
		escaped = strings.Replace(escaped, string(symbol), " ", -1)
	}
	return
}
