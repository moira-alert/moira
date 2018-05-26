package plotting

import (
	"sort"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

// TestSortByLen tests simple string array sorting by length
func TestSortByLen(t *testing.T) {
	labelsUnsorted := []string{
		"CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
		"ServiceName.HostName.CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
		"MetricPrefix.ServiceName.HostName.CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
		"CategoryCounterType.MetricName",
		"MetricName",
		"CategoryCounterName.CategoryCounterType.MetricName",
		"HostName.CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
	}
	labelsSorted := []string{
		"MetricName",
		"CategoryCounterType.MetricName",
		"CategoryCounterName.CategoryCounterType.MetricName",
		"CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
		"HostName.CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
		"ServiceName.HostName.CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
		"MetricPrefix.ServiceName.HostName.CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
	}
	Convey("Sort initial unsorted string array", t, func() {
		sort.Sort(SortedByLen(labelsUnsorted))
		So(len(labelsUnsorted), ShouldEqual, len(labelsSorted))
		So(labelsUnsorted, ShouldResemble, labelsSorted)
	})
}

// TestInt32ToTime tests simple int32 timestamp to time.Time converter
func TestInt32ToTime(t *testing.T) {
	int32timeStamp := int32(1527330278)
	humanReadabletimeStamp := time.Date(2018, 5, 26, 10, 24, 38, 0, time.UTC)
	Convey("Convert int32 timestamp into datetime", t, func() {
		converted := Int32ToTime(int32timeStamp)
		So(converted, ShouldResemble, humanReadabletimeStamp)
	})
	Convey("Convert int32 timestamp + 1 minute into datetime", t, func() {
		int32timeStamp += 60
		converted := Int32ToTime(int32timeStamp)
		So(converted, ShouldResemble, humanReadabletimeStamp.Add(time.Minute))
	})
}

// TestSanitizeLabelName tests simple label names shortener
func TestSanitizeLabelName(t *testing.T) {
	labelsCompleteForm := []string{
		"MetricName",
		"CategoryCounterType.MetricName",
		"CategoryCounterName.CategoryCounterType.MetricName",
		"CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
		"HostName.CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
		"ServiceName.HostName.CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
		"MetricPrefix.ServiceName.HostName.CategoryName.CategoryCounterName.CategoryCounterType.MetricName",
	}
	labelsShortForm := []string{
		"MetricName",
		"CategoryCounterType.MetricName",
		"CategoryCounterName.Categor...",
		"CategoryName.CategoryCounte...",
		"HostName.CategoryName.Categ...",
		"ServiceName.HostName.Catego...",
		"MetricPrefix.ServiceName.Ho...",
	}
	Convey("sanitize lables names", t, func() {
		maxLabelLength := 30
		shortLablelsList := make([]string, 0)
		for _, label := range labelsCompleteForm {
			shortLabel := SanitizeLabelName(label, maxLabelLength)
			shortLablelsList = append(shortLablelsList, shortLabel)
		}
		So(len(shortLablelsList), ShouldEqual, len(labelsShortForm))
		So(shortLablelsList, ShouldResemble, labelsShortForm)
	})
}

// TestFloatToHumanizedValueFormatter tests custom value formatter based on go-humanize library
func TestFloatToHumanizedValueFormatter(t *testing.T) {
	metricValues := []float64{
		999,
		1000,
		1000000,
		1000000000,
		1000000000000,
	}
	metricValuesFormatted := []string{
		"999",
		"1.00 K",
		"1.00 M",
		"1.00 G",
		"1.00 T",
	}
	Convey("format metric values", t, func() {
		formattedValues := make([]string, 0)
		for _, metricValue := range metricValues {
			formattedMetricValue := FloatToHumanizedValueFormatter(metricValue)
			formattedValues = append(formattedValues, formattedMetricValue)
		}
		So(len(formattedValues), ShouldEqual, len(metricValuesFormatted))
		So(formattedValues, ShouldResemble, metricValuesFormatted)
	})
}

// TestGetYAxisValuesFormatter tests all metric values will be formatted respectively with resolved plot limits
func TestGetYAxisValuesFormatter(t *testing.T) {
	lowLimits := Limits{
		Lowest:  0,
		Highest: 10,
	}
	mediumLimits := Limits{
		Lowest:  -10,
		Highest: 10,
	}
	highLimits := Limits{
		Lowest:  -1000,
		Highest: 1000,
	}
	Convey("format metric values with low limits", t, func() {
		metricValues := []float64{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		}
		formattedMetricValues := []string{
			"0.00", "1.00", "2.00", "3.00", "4.00", "5.00", "6.00", "7.00", "8.00", "9.00", "10.00",
		}
		valueFormatter := GetYAxisValuesFormatter(lowLimits)
		formattedValues := make([]string, 0)
		for _, metricValue := range metricValues {
			formattedValue := valueFormatter(metricValue)
			formattedValues = append(formattedValues, formattedValue)
		}
		So(len(formattedValues), ShouldEqual, len(formattedMetricValues))
		So(formattedValues, ShouldResemble, formattedMetricValues)
	})
	Convey("format metric values with medium limits", t, func() {
		metricValues := []float64{
			-5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5,
		}
		formattedMetricValues := []string{
			"-5", "-4", "-3", "-2", "-1", "0", "1", "2", "3", "4", "5",
		}
		valueFormatter := GetYAxisValuesFormatter(mediumLimits)
		formattedValues := make([]string, 0)
		for _, metricValue := range metricValues {
			formattedValue := valueFormatter(metricValue)
			formattedValues = append(formattedValues, formattedValue)
		}
		So(len(formattedValues), ShouldEqual, len(formattedMetricValues))
		So(formattedValues, ShouldResemble, formattedMetricValues)
	})
	Convey("format metric values with high limits", t, func() {
		metricValues := []float64{
			-1000, -100, 0, 100, 1000,
		}
		formattedMetricValues := []string{
			"-1.00 K", "-100", "0", "100", "1.00 K",
		}
		valueFormatter := GetYAxisValuesFormatter(highLimits)
		formattedValues := make([]string, 0)
		for _, metricValue := range metricValues {
			formattedValue := valueFormatter(metricValue)
			formattedValues = append(formattedValues, formattedValue)
		}
		So(len(formattedValues), ShouldEqual, len(formattedMetricValues))
		So(formattedValues, ShouldResemble, formattedMetricValues)
	})
}
