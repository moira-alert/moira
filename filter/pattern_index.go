package filter

import (
	"strings"

	"github.com/moira-alert/moira"

	"github.com/cespare/xxhash/v2"
)

// PatternNode contains pattern node
type PatternNode struct {
	Children   []*PatternNode
	Part       string
	Hash       uint64
	Prefix     string
	InnerParts []string
	Terminal   bool
}

// PatternIndex helps to index patterns and allows to match them by metric
type PatternIndex struct {
	Root     *PatternNode
	Logger   moira.Logger
	tierPool *tierPool
}

// NewPatternIndex creates new PatternIndex using patterns
func NewPatternIndex(logger moira.Logger, patterns []string, tierPool *tierPool) *PatternIndex {
	root := &PatternNode{}

	for _, pattern := range patterns {
		currentNode := root
		parts := strings.Split(pattern, ".")
		if hasEmptyParts(parts) {
			logger.Warningf("Pattern %s is ignored because it contains an empty part", pattern)
			continue
		}

		for i, part := range parts {
			found := false
			for _, child := range currentNode.Children {
				if part == child.Part {
					currentNode = child
					found = true
					break
				}
			}
			if !found {
				newNode := &PatternNode{Part: part}

				if currentNode.Prefix == "" {
					newNode.Prefix = part
				} else {
					newNode.Prefix = currentNode.Prefix + "." + part
				}

				if part == "*" || !strings.ContainsAny(part, "{*?") {
					newNode.Hash = xxhash.Sum64String(part)
				} else {
					if strings.Contains(part, "{") && strings.Contains(part, "}") {
						prefix, bigSuffix := split2(part, "{")
						inner, suffix := split2(bigSuffix, "}")
						innerParts := strings.Split(inner, ",")

						newNode.InnerParts = make([]string, 0, len(innerParts))
						for _, innerPart := range innerParts {
							whole := prefix + innerPart + suffix
							newNode.InnerParts = append(newNode.InnerParts, whole)
						}
					} else {
						newNode.InnerParts = []string{part}
					}
				}
				currentNode.Children = append(currentNode.Children, newNode)
				currentNode = newNode
			}
			if i == len(parts)-1 {
				currentNode.Terminal = true
			}
		}
	}

	return &PatternIndex{Logger: logger, Root: root, tierPool: tierPool}
}

// MatchPatterns allows to match pattern by metric
func (source *PatternIndex) MatchPatterns(metric string) []string {
	var (
		found int
		index int
		tier  *tier
	)

	// this is better be used on go>1.14,
	// because earlier go versions have
	// poor defer implementation
	tier = source.tierPool.acquireTier()
	defer source.tierPool.releaseTier(tier)

	tier.curr = append(tier.curr, source.Root)
	for i, c := range metric {
		if c == '.' {
			part := metric[index:i]
			if len(part) == 0 {
				return []string{}
			}
			index = i + 1

			if tier.findPart(part) == 0 {
				return []string{}
			}
		}
	}

	if found = tier.findPart(metric[index:]); found == 0 {
		return []string{}
	}

	matched := make([]string, 0, found)
	for _, node := range tier.curr {
		if node.Terminal {
			matched = append(matched, node.Prefix)
		}
	}
	return matched
}

func split2(s, sep string) (string, string) {
	splitResult := strings.SplitN(s, sep, 2)
	if len(splitResult) < 2 { //nolint
		return splitResult[0], ""
	}
	return splitResult[0], splitResult[1]
}

func hasEmptyParts(parts []string) bool {
	for _, part := range parts {
		if part == "" {
			return true
		}
	}
	return false
}
