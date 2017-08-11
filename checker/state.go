package checker

const (
	OK        = "OK"
	WARN      = "WARN"
	ERROR     = "ERROR"
	NODATA    = "NODATA"
	EXCEPTION = "EXCEPTION"
	DEL       = "DEL"
)

var scores = map[string]int64{
	OK:        0,
	DEL:       0,
	WARN:      1,
	ERROR:     100,
	NODATA:    1000,
	EXCEPTION: 100000,
}

func toMetricState(state string) string {
	if state == DEL {
		return NODATA
	}
	return state
}
