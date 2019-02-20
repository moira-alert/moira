package filter

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics/graphite"
	"github.com/vova616/xxhash"
)

var asteriskHash = xxhash.Checksum32([]byte("*"))

// PatternStorage contains pattern tree
type PatternStorage struct {
	database    moira.Database
	metrics     *graphite.FilterMetrics
	logger      moira.Logger
	PatternTree atomic.Value
}

// PatternNode contains pattern node
type PatternNode struct {
	Children   []*PatternNode
	Part       string
	Hash       uint32
	Prefix     string
	InnerParts []string
}

// NewPatternStorage creates new PatternStorage struct
func NewPatternStorage(database moira.Database, metrics *graphite.FilterMetrics, logger moira.Logger) (*PatternStorage, error) {
	storage := &PatternStorage{
		database: database,
		metrics:  metrics,
		logger:   logger,
	}
	err := storage.RefreshTree()
	return storage, err
}

// RefreshTree builds pattern tree from redis data
func (storage *PatternStorage) RefreshTree() error {
	patterns, err := storage.database.GetPatterns()
	if err != nil {
		return err
	}
	return storage.buildTree(patterns)
}

// ProcessIncomingMetric validates, parses and matches incoming raw string
func (storage *PatternStorage) ProcessIncomingMetric(lineBytes []byte) *moira.MatchedMetric {
	storage.metrics.TotalMetricsReceived.Inc(1)
	count := storage.metrics.TotalMetricsReceived.Count()

	metric, value, timestamp, err := storage.parseMetric(lineBytes)
	if err != nil {
		storage.logger.Infof("cannot parse input: %v", err)
		return nil
	}

	storage.metrics.ValidMetricsReceived.Inc(1)

	matchingStart := time.Now()
	matched := storage.matchPattern(metric)
	if count%10 == 0 {
		storage.metrics.MatchingTimer.UpdateSince(matchingStart)
	}
	if len(matched) > 0 {
		storage.metrics.MatchingMetricsReceived.Inc(1)
		return &moira.MatchedMetric{
			Metric:             string(metric),
			Patterns:           matched,
			Value:              value,
			Timestamp:          timestamp,
			RetentionTimestamp: timestamp,
			Retention:          60,
		}
	}
	return nil
}

// matchPattern returns array of matched patterns
func (storage *PatternStorage) matchPattern(metric []byte) []string {
	currentLevel := []*PatternNode{storage.PatternTree.Load().(*PatternNode)}
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

// parseMetric parses metric from string
// supported format: "<metricString> <valueFloat64> <timestampInt64>"
func (*PatternStorage) parseMetric(input []byte) ([]byte, float64, int64, error) {
	firstSpaceIndex := bytes.IndexByte(input, ' ')
	if firstSpaceIndex < 1 {
		return nil, 0, 0, fmt.Errorf("too few space-separated items: '%s'", input)
	}

	secondSpaceIndex := bytes.IndexByte(input[firstSpaceIndex+1:], ' ')
	if secondSpaceIndex < 1 {
		return nil, 0, 0, fmt.Errorf("too few space-separated items: '%s'", input)
	}
	secondSpaceIndex += firstSpaceIndex + 1

	metric := input[:firstSpaceIndex]
	if !isPrintableASCII(metric) {
		return nil, 0, 0, fmt.Errorf("non-ascii or non-printable chars in metric name: '%s'", input)
	}

	value, err := strconv.ParseFloat(unsafeString(input[firstSpaceIndex+1:secondSpaceIndex]), 64)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("cannot parse value: '%s' (%s)", input, err)
	}

	timestamp, err := parseTimestamp(unsafeString(input[secondSpaceIndex+1:]))
	if err != nil || timestamp == 0 {
		return nil, 0, 0, fmt.Errorf("cannot parse timestamp: '%s' (%s)", input, err)
	}

	return metric, value, timestamp, nil
}

func (storage *PatternStorage) buildTree(patterns []string) error {
	newTree := &PatternNode{}

	for _, pattern := range patterns {
		currentNode := newTree
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

	storage.PatternTree.Store(newTree)
	return nil
}

func parseTimestamp(unixTimestamp string) (int64, error) {
	timestamp, err := strconv.ParseFloat(unixTimestamp, 64)
	return int64(timestamp), err
}

func hasEmptyParts(parts []string) bool {
	for _, part := range parts {
		if part == "" {
			return true
		}
	}
	return false
}

func findPart(part []byte, currentLevel []*PatternNode) ([]*PatternNode, int) {
	nextLevel := make([]*PatternNode, 0, 64)
	hash := xxhash.Checksum32(part)
	for _, node := range currentLevel {
		for _, child := range node.Children {
			match := false

			if child.Hash == asteriskHash || child.Hash == hash {
				match = true
			} else if len(child.InnerParts) > 0 {
				for _, innerPart := range child.InnerParts {
					innerMatch, _ := path.Match(innerPart, string(part))
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

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func isPrintableASCII(b []byte) bool {
	for i := 0; i < len(b); i++ {
		if b[i] < 0x20 || b[i] > 0x7E {
			return false
		}
	}

	return true
}
