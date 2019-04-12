package plotting

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/moira-alert/moira"
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
		sort.Sort(sortedByLen(labelsUnsorted))
		So(len(labelsUnsorted), ShouldEqual, len(labelsSorted))
		So(labelsUnsorted, ShouldResemble, labelsSorted)
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
		"Рост количества ответов nginx р95",
		"ЯдлиннаядлиннаястрокабезпробеловизрусскихбуквчтобыТимуриАркадийневыпендривалисьаЛешенепришлосьприходитьивсеобъяснятьнормально😈",
		"Привет, не режь меня!",
	}
	labelsShortForm := []string{
		"MetricName",
		"CategoryCounterType.MetricName",
		"CategoryCounterName.Categor...",
		"CategoryName.CategoryCounte...",
		"HostName.CategoryName.Categ...",
		"ServiceName.HostName.Catego...",
		"MetricPrefix.ServiceName.Ho...",
		"Рост количества ответов ngi...",
		"Ядлиннаядлиннаястрокабезпро...",
		"Привет, не режь меня!",
	}
	Convey("sanitize lables names", t, func() {
		maxLabelLength := 30
		shortLablelsList := make([]string, 0)
		for _, label := range labelsCompleteForm {
			shortLabel := sanitizeLabelName(label, maxLabelLength)
			shortLablelsList = append(shortLablelsList, shortLabel)
		}
		So(len(shortLablelsList), ShouldEqual, len(labelsShortForm))
		So(shortLablelsList, ShouldResemble, labelsShortForm)
	})
}

// TestPercentsOfRange is a simple test of percentsOfRange method
func TestPercentsOfRange(t *testing.T) {
	Convey("Test nth percent is calculated correctly", t, func() {
		for i := 0; i < 100; i++ {
			expected := i
			actual := percentsOfRange(float64(0), float64(100), float64(i))
			So(actual, ShouldEqual, expected)
		}
	})
}

// TestTimeValueFormatter tests time.Time to formatted string converter
func TestTimeValueFormatter(t *testing.T) {
	dateTimeFormat, separator := "15:04", ":"
	timeValue := moira.Int64ToTime(int64(1527330278))
	locationIncrements := map[string]int{
		"Europe/Moscow":      3,
		"Asia/Yekaterinburg": 5,
	}
	Convey("Format int64 timestamps into correct strings", t, func() {
		for name, increment := range locationIncrements {
			location, _ := time.LoadLocation(name)
			storage := &locationStorage{location: location}
			formatted := storage.formatTimeWithLocation(timeValue, dateTimeFormat)
			formattedHourAndMinute := strings.Split(formatted, separator)
			formattedHour, _ := strconv.Atoi(formattedHourAndMinute[0])
			formattedMinute, _ := strconv.Atoi(formattedHourAndMinute[1])
			fmt.Printf("%s: %s,\n%s: %s\n\n",
				timeValue.Location().String(), timeValue.String(), location.String(), formatted)
			So(formattedMinute, ShouldEqual, timeValue.Minute())
			So(formattedHour, ShouldEqual, timeValue.Add(time.Duration(increment)*time.Hour).Hour())
		}
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
			formattedMetricValue := floatToHumanizedValueFormatter(metricValue)
			formattedValues = append(formattedValues, formattedMetricValue)
		}
		So(len(formattedValues), ShouldEqual, len(metricValuesFormatted))
		So(formattedValues, ShouldResemble, metricValuesFormatted)
	})
}

// TestGetYAxisValuesFormatter tests all metric values will be formatted respectively with resolved plot limits
func TestGetYAxisValuesFormatter(t *testing.T) {
	lowLimits := plotLimits{
		lowest:  0,
		highest: 10,
	}
	mediumLimits := plotLimits{
		lowest:  -10,
		highest: 10,
	}
	highLimits := plotLimits{
		lowest:  -1000,
		highest: 1000,
	}
	Convey("format metric values with low limits", t, func() {
		metricValues := []float64{
			0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		}
		formattedMetricValues := []string{
			"0.00", "1.00", "2.00", "3.00", "4.00", "5.00", "6.00", "7.00", "8.00", "9.00", "10.00",
		}
		valueFormatter, maxMarkLen := getYAxisValuesFormatter(lowLimits)
		formattedValues := make([]string, 0)
		for _, metricValue := range metricValues {
			formattedValue := valueFormatter(metricValue)
			formattedValues = append(formattedValues, formattedValue)
		}
		So(maxMarkLen, ShouldEqual, 5)
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
		valueFormatter, maxMarkLen := getYAxisValuesFormatter(mediumLimits)
		formattedValues := make([]string, 0)
		for _, metricValue := range metricValues {
			formattedValue := valueFormatter(metricValue)
			formattedValues = append(formattedValues, formattedValue)
		}
		So(maxMarkLen, ShouldEqual, 3)
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
		valueFormatter, maxMarkLen := getYAxisValuesFormatter(highLimits)
		formattedValues := make([]string, 0)
		for _, metricValue := range metricValues {
			formattedValue := valueFormatter(metricValue)
			formattedValues = append(formattedValues, formattedValue)
		}
		So(maxMarkLen, ShouldEqual, 7)
		So(len(formattedValues), ShouldEqual, len(formattedMetricValues))
		So(formattedValues, ShouldResemble, formattedMetricValues)
	})
}
