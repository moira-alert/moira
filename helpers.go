package moira

// UseString gets pointer value of string or default string if pointer is nil
func UseString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

// UseFloat64 gets pointer value of float64 or default float64 if pointer is nil
func UseFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// Subset return whether first is a subset of second
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

// LeftJoinStrings return sublist of strings presented in left list, but not in right list
func LeftJoinStrings(left, right []string) []string {
	rightValues := make(map[string]bool)
	for _, value := range right {
		rightValues[value] = true
	}
	arr := make([]string, 0)
	for _, leftValue := range left {
		if _, ok := rightValues[leftValue]; !ok {
			arr = append(arr, leftValue)
		}
	}
	return arr
}

// LeftJoinTriggers return sublist of moira.Triggers presented in left list, but not in right list
func LeftJoinTriggers(left, right []*Trigger) []*Trigger {
	rightValues := make(map[string]bool)
	for _, value := range right {
		rightValues[value.ID] = true
	}
	arr := make([]*Trigger, 0)
	for _, leftValue := range left {
		if _, ok := rightValues[leftValue.ID]; !ok {
			arr = append(arr, leftValue)
		}
	}
	return arr
}
