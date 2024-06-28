package filter

import (
	"testing"

	"github.com/moira-alert/moira"
)

var (
	metric = "test.name;testTag1=test1test1;testTag2=test2test2test2;testTag3=test3test4test5;testTag4=test6test7test8test9"
	name   = "test.metric"

	labels = map[string]string{
		"test1": "test1test1",
		"test2": "test2test2test2",
		"test3": "test3test3test3test3",
		"test4": "test4test4test4test4test4",
		"test5": "test5test5test5test5",
		"test6": "test6test6test6",
		"test7": "test7test7",
		"test8": "test8",
	}
)

func BenchmarkRestoreMetricStringByNameAndLabels(b *testing.B) {
	metric := name
	for key, value := range labels {
		metric += ";" + key + "=" + value
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = restoreMetricStringByNameAndLabels(name, labels, len(metric))
	}
}

func BenchmarkParseNameAndLabels(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := parseNameAndLabels(moira.UnsafeStringToBytes(metric))
		if err != nil {
			b.Fatal(err)
		}
	}
}
