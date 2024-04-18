package filter

import (
	"path"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/moira-alert/moira"
)

var asteriskHash = xxhash.Sum64String("*")

type PrefixTree struct {
	Root   *PatternNode
	Logger moira.Logger
}

// PatternNode contains pattern node.
type PatternNode struct {
	Children   []*PatternNode
	Part       string
	Hash       uint64
	Prefix     string
	InnerParts []string
	Terminal   bool
	Payload    map[string]MatchingHandler
}

// Add inserts pattern in tree.
func (source *PrefixTree) Add(pattern string) {
	source.AddWithPayload(pattern, "", nil)
}

// AddWithPayload inserts pattern and payload in tree.
func (source *PrefixTree) AddWithPayload(pattern string, payloadKey string, payloadValue MatchingHandler) {
	currentNode := source.Root
	parts := strings.Split(pattern, ".")
	if hasEmptyParts(parts) {
		source.Logger.Warning().
			String("pattern", pattern).
			Msg("Pattern is ignored because it contains an empty part")
		return
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
			if payloadValue != nil {
				if currentNode.Payload == nil {
					currentNode.Payload = make(map[string]MatchingHandler)
				}
				currentNode.Payload[payloadKey] = payloadValue
			}
		}
	}
}

// Match finds metric in tree and returns prefixes for all matched nodes.
func (source *PrefixTree) Match(metric string) []string {
	nodes, found := source.findNodes(metric)
	if found == 0 {
		return []string{}
	}

	matched := make([]string, 0, found)
	for _, node := range nodes {
		if node.Terminal {
			matched = append(matched, node.Prefix)
		}
	}

	return matched
}

// MatchWithValue finds metric in tree and returns payloads for all matched nodes.
func (source *PrefixTree) MatchWithValue(metric string) map[string]MatchingHandler {
	nodes, _ := source.findNodes(metric)
	if nodes == nil {
		return map[string]MatchingHandler{}
	}

	matched := make(map[string]MatchingHandler)
	for _, node := range nodes {
		if node.Terminal {
			if node.Payload == nil {
				matched[node.Prefix] = nil
			}

			for pattern, matchingHandler := range node.Payload {
				matched[pattern] = matchingHandler
			}
		}
	}

	return matched
}

func (source *PrefixTree) findNodes(metric string) ([]*PatternNode, int) {
	currentLevel := []*PatternNode{source.Root}
	var found, index int
	for i, c := range metric {
		if c == '.' {
			part := metric[index:i]

			if len(part) == 0 {
				source.Logger.Warning().
					String("metric", metric).
					Msg("Metric is ignored, because it contains empty parts")
				return nil, 0
			}

			index = i + 1

			currentLevel, found = findPart(part, currentLevel)
			if found == 0 {
				return nil, 0
			}
		}
	}

	part := metric[index:]
	currentLevel, found = findPart(part, currentLevel)
	if found == 0 {
		return nil, 0
	}

	return currentLevel, found
}

func findPart(part string, currentLevel []*PatternNode) ([]*PatternNode, int) {
	nextLevel := make([]*PatternNode, 0, 5)

	hash := xxhash.Sum64String(part)
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
