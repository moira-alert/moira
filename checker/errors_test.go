package checker

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestErrUnexpectedAloneMetric_Error(t *testing.T) {
	type fields struct {
		expected map[string]bool
		actual   map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "expected is empty and actual is not",
			fields: fields{
				expected: map[string]bool{},
				actual: map[string]string{
					"t1": "metric.test.1",
				},
			},
			want: strings.ReplaceAll(`Unexpected to have some targets with only one metric.
			Expected targets with only one metric:
			Actual targets with only one metric:
				t1 — metric.test.1

			Probably you want to set "Single" flag for following targets: t1`, "\n\t\t\t", "\n"),
		},
		{
			name: "expected is not empty and actual is empty",
			fields: fields{
				expected: map[string]bool{
					"t1": true,
				},
				actual: map[string]string{},
			},
			want: strings.ReplaceAll(`Unexpected to have some targets with only one metric.
			Expected targets with only one metric: t1
			Actual targets with only one metric:

			Probably you want to switch off "Single" flag for following targets: t1`, "\n\t\t\t", "\n"),
		},
		{
			name: "expected  and actual are not empty",
			fields: fields{
				expected: map[string]bool{
					"t1": true,
				},
				actual: map[string]string{
					"t2": "metric.test.1",
				},
			},
			want: strings.ReplaceAll(`Unexpected to have some targets with only one metric.
			Expected targets with only one metric: t1
			Actual targets with only one metric:
				t2 — metric.test.1

			Probably you want to set "Single" flag for following targets: t2

			Probably you want to switch off "Single" flag for following targets: t1`, "\n\t\t\t", "\n"),
		},
		{
			name: "check sorting",
			fields: fields{
				expected: map[string]bool{
					"t1": true,
					"t2": true,
					"t3": true,
					"t4": true,
					"t5": true,
				},
				actual: map[string]string{
					"t6":  "metric.test.1",
					"t7":  "metric.test.2",
					"t8":  "metric.test.3",
					"t9":  "metric.test.4",
					"t10": "metric.test.5",
				},
			},
			want: strings.ReplaceAll(`Unexpected to have some targets with only one metric.
			Expected targets with only one metric: t1, t2, t3, t4, t5
			Actual targets with only one metric:
				t10 — metric.test.5
				t6 — metric.test.1
				t7 — metric.test.2
				t8 — metric.test.3
				t9 — metric.test.4

			Probably you want to set "Single" flag for following targets: t10, t6, t7, t8, t9

			Probably you want to switch off "Single" flag for following targets: t1, t2, t3, t4, t5`, "\n\t\t\t", "\n"),
		},
	}
	Convey("ErrUnexpectedAloneMetric message", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				err := ErrUnexpectedAloneMetric{
					expected: tt.fields.expected,
					actual:   tt.fields.actual,
				}
				So(err.Error(), ShouldResemble, tt.want)
			})
		}
	})
}
