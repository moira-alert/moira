package filter

import (
	"path"
	"sync"

	"github.com/cespare/xxhash/v2"
)

const (
	// it is supposed that this many PatternNode`s
	// are allocated twice per worker
	// (one for current tree level and one for the next one)
	// this constant might need adjustment
	tierAlloc = 32
)

var (
	asteriskHash = xxhash.Sum64String("*")
)

// tier wraps 2 pattern tree levels adjacent to each other
type tier struct {
	curr []*PatternNode
	next []*PatternNode
}

// tierPool uses sync.Pool to reuse tier`s
// in order to avoid redundant allocation
type tierPool struct {
	pool sync.Pool
}

func newTier() *tier {
	return &tier{
		curr: make([]*PatternNode, 0, tierAlloc),
		next: make([]*PatternNode, 0, tierAlloc),
	}
}

// findPart seeks for the given part among curr nodes level
// while filling up the next level
// it is assumed that curr is filled by the caller
// or during the previous call of findPart
func (t *tier) findPart(part string) int {
	hash := xxhash.Sum64String(part)
	match := false

	for _, node := range t.curr {
		for _, child := range node.Children {
			match = false

			if child.Hash == asteriskHash || child.Hash == hash {
				match = true
			} else if len(child.InnerParts) > 0 {
				for _, innerPart := range child.InnerParts {
					innerMatch, _ := path.Match(innerPart, part)
					if innerMatch {
						match = true
						break
					}
				}
			}

			if match {
				t.next = append(t.next, child)
			}
		}
	}

	// resulting level (next) is moved to curr in case there will be the next iteration
	t.curr, t.next = t.next, t.curr[:0]

	// matched nodes quantity as a result
	return len(t.curr)
}

func newTierPool() *tierPool {
	return &tierPool{pool: sync.Pool{
		New: func() interface{} {
			return newTier()
		},
	}}
}

func (tp *tierPool) acquireTier() *tier {
	return tp.pool.Get().(*tier)
}

func (tp *tierPool) releaseTier(t *tier) {
	// x = x[:0] truncates slice leaving no elements in it
	// but keeps the capacity of the underlying buffer
	t.curr = t.curr[:0]
	t.next = t.next[:0]
	tp.pool.Put(t)
}
