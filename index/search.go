package index

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
)

const (
	indexWaitTimeout = time.Second * 3
)

// SearchTriggers search for triggers in index and returns slice of trigger IDs
func (index *Index) SearchTriggers(filterTags []string, searchString string, onlyErrors bool, page int64, size int64) (searchResults []*moira.SearchResult, total int64, err error) {
	if !index.checkIfIndexIsReady() {
		return make([]*moira.SearchResult, 0), 0, fmt.Errorf("index is not ready, please try later")
	}
	return index.triggerIndex.Search(filterTags, searchString, onlyErrors, page, size)
}

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
