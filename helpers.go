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

func UseInt64(i *int64) int64{
	if i == nil {
		return 0
	}
	return *i
}
