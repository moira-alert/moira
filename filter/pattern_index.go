package filter

import (
	"fmt"
	"path"
	"strings"

	"github.com/moira-alert/moira"

	"github.com/vova616/xxhash"
)

var asteriskHash = xxhash.Checksum32([]byte("*"))

//PatternNode contains pattern node
type PatternNode struct {
	Children   []*PatternNode
	Part       string
	Hash       uint32
	Prefix     string
	InnerParts []string
}

// PatternIndex helps to index patterns and allows to match them by metric
type PatternIndex struct {
	Root *PatternNode
}

// NewPatternIndex creates new PatternIndex using patterns
func NewPatternIndex(patterns []string) *PatternIndex {
	root := &PatternNode{}

	for _, pattern := range patterns {
		currentNode := root
		parts := strings.Split(pattern, ".")
		if hasEmptyParts(parts) {
			continue
		}
		for _, part := range parts {
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
					newNode.Prefix = fmt.Sprintf("%s.%s", currentNode.Prefix, part)
				}

				if part == "*" || !strings.ContainsAny(part, "{*?") {
					newNode.Hash = xxhash.Checksum32([]byte(part))
				} else {
					if strings.Contains(part, "{") && strings.Contains(part, "}") {
						prefix, bigSuffix := split2(part, "{")
						inner, suffix := split2(bigSuffix, "}")
						innerParts := strings.Split(inner, ",")

						newNode.InnerParts = make([]string, 0, len(innerParts))
						for _, innerPart := range innerParts {
							newNode.InnerParts = append(newNode.InnerParts, fmt.Sprintf("%s%s%s", prefix, innerPart, suffix))
						}
					} else {
						newNode.InnerParts = []string{part}
					}

				}
				currentNode.Children = append(currentNode.Children, newNode)
				currentNode = newNode
			}
		}
	}

	return &PatternIndex{Root: root}
}

// MatchPatterns allows to match pattern by metric
func (source *PatternIndex) MatchPatterns(metric string) []string {
	currentLevel := []*PatternNode{source.Root}
	var found, index int
	for i, c := range metric {
		if c == '.' {
			part := metric[index:i]

			if len(part) == 0 {
				return []string{}
			}

			index = i + 1

			currentLevel, found = findPart(part, currentLevel)
			if found == 0 {
				return []string{}
			}
		}
	}

	part := metric[index:]
	currentLevel, found = findPart(part, currentLevel)
	if found == 0 {
		return []string{}
	}

	matched := make([]string, 0, found)
	for _, node := range currentLevel {
		if len(node.Children) == 0 {
			matched = append(matched, node.Prefix)
		}
	}

	return matched
}

func findPart(part string, currentLevel []*PatternNode) ([]*PatternNode, int) {
	nextLevel := make([]*PatternNode, 0, 64)
	hash := xxhash.Checksum32(moira.UnsafeStringToBytes(part))
	for _, node := range currentLevel {
		for _, child := range node.Children {
			match := false

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
				nextLevel = append(nextLevel, child)
			}
		}
	}
	return nextLevel, len(nextLevel)
}

func split2(s, sep string) (string, string) {
	splitResult := strings.SplitN(s, sep, 2)
	if len(splitResult) < 2 {
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
