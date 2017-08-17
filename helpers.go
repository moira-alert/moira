package moira

func UseString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

func UseFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
