package cache

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/metrics/graphite"
	"github.com/vova616/xxhash"
	"log"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
)

var asteriskHash = xxhash.Checksum32([]byte("*"))

// PatternStorage contains pattern tree
type PatternStorage struct {
	database       moira.Database
	metrics        *graphite.CacheMetrics
	logger         moira.Logger
	PatternTree    *patternNode
	logParseErrors bool
}

// patternNode contains pattern node
type patternNode struct {
	Children   []*patternNode
	Part       string
	Hash       uint32
	Prefix     string
	InnerParts []string
}

// NewPatternStorage creates new PatternStorage struct
func NewPatternStorage(database moira.Database, metrics *graphite.CacheMetrics, logger moira.Logger, logParseErrors bool) (*PatternStorage, error) {
	storage := &PatternStorage{
		database:       database,
		metrics:        metrics,
		logger:         logger,
		logParseErrors: logParseErrors,
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
	count := atomic.AddInt64(&storage.metrics.TotalReceived, 1)

	metric, value, timestamp, err := storage.parseMetricFromString(lineBytes)
	if err != nil {
		if storage.logParseErrors {
			log.Printf("cannot parse input: %s", err)
		}

		return nil
	}

	atomic.AddInt64(&storage.metrics.ValidReceived, 1)

	matchingStart := time.Now()
	matched := storage.matchPattern(metric)
	if count%10 == 0 {
		storage.metrics.MatchingTimer.UpdateSince(matchingStart)
	}
	if len(matched) > 0 {
		atomic.AddInt64(&storage.metrics.MatchedReceived, 1)
		return &moira.MatchedMetric{string(metric), matched, value, timestamp, timestamp, 60}
	}
	return nil
}

// matchPattern returns array of matched patterns
func (storage *PatternStorage) matchPattern(metric []byte) []string {
	currentLevel := []*patternNode{storage.PatternTree}
	found := 0
	index := 0
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

	part := metric[index:len(metric)]
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

// parseMetricFromString parses metric from string
// supported format: "<metricString> <valueFloat64> <timestampInt64>"
func (storage *PatternStorage) parseMetricFromString(line []byte) ([]byte, float64, int64, error) {
	var parts [3][]byte
	partIndex := 0
	partOffset := 0
	for i, b := range line {
		r := rune(b)
		if r > unicode.MaxASCII || !strconv.IsPrint(r) {
			return nil, 0, 0, fmt.Errorf("non-ascii or non-printable chars in metric name: '%s'", line)
		}
		if b == ' ' {
			parts[partIndex] = line[partOffset:i]
			partOffset = i + 1
			partIndex++
		}
		if partIndex > 2 {
			return nil, 0, 0, fmt.Errorf("too many space-separated items: '%s'", line)
		}
	}

	if partIndex < 2 {
		return nil, 0, 0, fmt.Errorf("too few space-separated items: '%s'", line)
	}

	parts[partIndex] = line[partOffset:]

	metric := parts[0]
	if len(metric) < 1 {
		return nil, 0, 0, fmt.Errorf("metric name is empty: '%s'", line)
	}

	value, err := strconv.ParseFloat(string(parts[1]), 64)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("cannot parse value: '%s' (%s)", line, err)
	}

	timestamp, err := strconv.ParseInt(string(parts[2]), 10, 64)
	if err != nil || timestamp == 0 {
		return nil, 0, 0, fmt.Errorf("cannot parse timestamp: '%s' (%s)", line, err)
	}

	return metric, value, timestamp, nil
}

func (storage *PatternStorage) buildTree(patterns []string) error {
	newTree := &patternNode{}

	for _, pattern := range patterns {
		currentNode := newTree
		parts := strings.Split(pattern, ".")
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
				newNode := &patternNode{Part: part}

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

	storage.PatternTree = newTree
	return nil
}

func findPart(part []byte, currentLevel []*patternNode) ([]*patternNode, int) {
	nextLevel := make([]*patternNode, 0, 64)
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
