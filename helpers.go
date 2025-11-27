package moira

import (
	"bytes"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// BytesScanner allows to scan for subslices separated by separator.
type BytesScanner struct {
	source         []byte
	index          int
	separator      byte
	emitEmptySlice bool
}

// Map applies a transformation function to each element in the input slice and returns a new slice with the transformed elements.
func Map[T any, R any](input []T, transform func(T) R) []R {
	result := make([]R, len(input))
	for i, v := range input {
		result[i] = transform(v)
	}

	return result
}

// HasNext checks if next subslice available or not.
func (it *BytesScanner) HasNext() bool {
	return it.index < len(it.source) || it.emitEmptySlice
}

// Next returns available subslice and advances the scanner to next slice.
func (it *BytesScanner) Next() (result []byte) {
	if it.emitEmptySlice {
		it.emitEmptySlice = false
		result = make([]byte, 0)

		return result
	}

	scannerIndex := it.index
	separatorIndex := bytes.IndexByte(it.source[scannerIndex:], it.separator)

	if separatorIndex < 0 {
		result = it.source[scannerIndex:]
		it.index = len(it.source)
	} else {
		separatorIndex += scannerIndex
		result = it.source[scannerIndex:separatorIndex]

		if separatorIndex == len(it.source)-1 {
			it.emitEmptySlice = true
		}

		it.index = separatorIndex + 1
	}

	return result
}

// NewBytesScanner slices bytes into all subslices separated by separator and returns a scanner
// which allows scanning for these subslices.
func NewBytesScanner(bytes []byte, separator byte) *BytesScanner {
	return &BytesScanner{
		source:         bytes,
		index:          0,
		separator:      separator,
		emitEmptySlice: false,
	}
}

// Int64ToTime returns time.Time from int64.
func Int64ToTime(timeStamp int64) time.Time {
	return time.Unix(timeStamp, 0).UTC()
}

// UseString gets pointer value of string or default string if pointer is nil.
func UseString(str *string) string {
	if str == nil {
		return ""
	}

	return *str
}

// UseFloat64 gets pointer value of float64 or default float64 if pointer is nil.
func UseFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}

	return *f
}

// IsFiniteNumber checks float64 for Inf and NaN. If it is then float64 is not valid.
func IsFiniteNumber(val float64) bool {
	return !(math.IsNaN(val) || math.IsInf(val, 0))
}

// Subset return whether first is a subset of second.
func Subset(first, second []string) bool {
	set := make(map[string]bool)
	for _, value := range second {
		set[value] = true
	}

	for _, value := range first {
		if !set[value] {
			return false
		}
	}

	return true
}

// Intersect returns the intersection of multiple arrays.
func Intersect[T comparable](lists ...[]T) []T {
	if len(lists) == 0 {
		return []T{}
	}

	intersection := make(map[T]bool)
	for _, value := range lists[0] {
		intersection[value] = true
	}

	for _, stringList := range lists[1:] {
		currentSet := make(map[T]bool)

		for _, value := range stringList {
			if intersection[value] {
				currentSet[value] = true
			}
		}

		intersection = currentSet
	}

	result := make([]T, 0, len(intersection))
	for value := range intersection {
		result = append(result, value)
	}

	return result
}

// SymmetricDiff returns the members of the set resulting from the difference between the first set and all the successive lists.
func SymmetricDiff[T comparable](lists ...[]T) []T {
	if len(lists) == 0 {
		return []T{}
	}

	allElements := make(map[T]bool)

	for _, list := range lists {
		for _, value := range list {
			allElements[value] = true
		}
	}

	intersection := Intersect(lists...)

	for _, value := range intersection {
		delete(allElements, value)
	}

	result := make([]T, 0, len(allElements))
	for value := range allElements {
		result = append(result, value)
	}

	return result
}

// GetStringListsDiff returns the members of the set resulting from the difference between the first set and all the successive lists.
func GetStringListsDiff(stringLists ...[]string) []string {
	if len(stringLists) == 0 {
		return []string{}
	}

	leftValues := make(map[string]bool)
	for _, value := range stringLists[0] {
		leftValues[value] = true
	}

	for _, stringList := range stringLists[1:] {
		for _, value := range stringList {
			delete(leftValues, value)
		}
	}

	result := make([]string, 0)

	for _, value := range stringLists[0] {
		if _, ok := leftValues[value]; ok {
			result = append(result, value)
		}
	}

	return result
}

// GetTriggerListsDiff returns the members of the set resulting from the difference between the first set and all the successive lists.
func GetTriggerListsDiff(triggerLists ...[]*Trigger) []*Trigger {
	if len(triggerLists) == 0 {
		return []*Trigger{}
	}

	leftValues := make(map[string]bool)

	for _, value := range triggerLists[0] {
		if value != nil {
			leftValues[value.ID] = true
		}
	}

	for _, triggerList := range triggerLists[1:] {
		for _, trigger := range triggerList {
			if trigger != nil {
				delete(leftValues, trigger.ID)
			}
		}
	}

	result := make([]*Trigger, 0)

	for _, value := range triggerLists[0] {
		if value == nil {
			continue
		}

		if _, ok := leftValues[value.ID]; ok {
			result = append(result, value)
		}
	}

	return result
}

// ChunkSlice gets slice of strings and chunks it to a given size. It returns a batch of chunked lists.
func ChunkSlice(original []string, chunkSize int) (divided [][]string) {
	if chunkSize < 1 {
		return divided
	}

	for i := 0; i < len(original); i += chunkSize {
		end := min(i+chunkSize, len(original))

		divided = append(divided, original[i:end])
	}

	return divided
}

func RoundToNearestRetention(ts, retention int64) int64 {
	return (ts + retention/2) / retention * retention
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

// ReplaceSubstring removes one substring between the beginning and end substrings and replaces it with a replaced.
func ReplaceSubstring(str, begin, end, replaced string) string {
	result := str

	startIndex := strings.Index(str, begin)
	if startIndex != -1 {
		startIndex += len(begin)
		endIndex := strings.Index(str[startIndex:], end)

		if endIndex != -1 {
			endIndex += len(str[:startIndex])
			result = str[:startIndex] + replaced + str[endIndex:]
		}
	}

	return result
}

type Comparable interface {
	Less(other Comparable) (bool, error)
}

// MergeToSorted Merge is a generic function that performs a merge of two sorted arrays into one sorted array.
func MergeToSorted[T Comparable](arr1, arr2 []T) ([]T, error) {
	merged := make([]T, 0, len(arr1)+len(arr2))
	i, j := 0, 0

	for i < len(arr1) && j < len(arr2) {
		less, err := arr1[i].Less(arr2[j])
		if err != nil {
			return nil, err
		}

		if less {
			merged = append(merged, arr1[i])
			i++
		} else {
			merged = append(merged, arr2[j])
			j++
		}
	}

	for i < len(arr1) {
		merged = append(merged, arr1[i])
		i++
	}

	for j < len(arr2) {
		merged = append(merged, arr2[j])
		j++
	}

	return merged, nil
}

// ValidateStruct is a default generic function that uses a validator to validate structure fields.
func ValidateStruct(s any) error {
	validator := validator.New()
	return validator.Struct(s)
}

// GetUniqueValues gets a collection and return unique items of collection in random order.
func GetUniqueValues[T comparable](objs ...T) []T {
	set := make(map[T]struct{})
	for _, obj := range objs {
		set[obj] = struct{}{}
	}

	res := make([]T, 0, len(set))
	for key := range set {
		res = append(res, key)
	}

	return res
}

// EqualTwoPointerValues checks that both pointers is not nil and if they both are not nil compares values
// that they are pointed to.
func EqualTwoPointerValues[T comparable](first, second *T) bool {
	if first != nil && second != nil {
		return *first == *second
	}

	return first == nil && second == nil
}

// ValidateURL returns error on invalid url.
func ValidateURL(requestUrl string) error {
	urlStruct, err := url.ParseRequestURI(requestUrl)
	if err != nil {
		return err
	}

	if urlStruct.Scheme != "http" && urlStruct.Scheme != "https" {
		return fmt.Errorf("bad url scheme: %s", urlStruct.Scheme)
	}

	if urlStruct.Host == "" {
		return fmt.Errorf("host is empty")
	}

	return nil
}

// CalculatePercentage computes the percentage of 'part' relative to 'total' as a uint8 pointer, or returns nil if invalid.
func CalculatePercentage(part, total uint64) *uint8 {
	if total == 0 {
		return nil
	}

	percentage := (float64(part) * float64(100)) / float64(total)
	if percentage > math.MaxUint8 {
		return nil
	}

	percentageValue := uint8(percentage)

	return &percentageValue
}

// SafeAdd safely adds two uint64 numbers and returns an error if an overflow occurs.
func SafeAdd(a, b uint64) (uint64, error) {
	result := a + b
	if result < a {
		return 0, fmt.Errorf("integer overflow occurred during addition")
	}

	return result, nil
}

// MapToSlice converts a map's values into a slice.
func MapToSlice[K, V comparable](input map[K]V) []V {
	res := make([]V, 0, len(input))
	for _, v := range input {
		res = append(res, v)
	}

	return res
}
