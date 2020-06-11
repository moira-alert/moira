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
			want: "Unexpected to have some targets with only one metric.\n" + "Expected targets with only one metric: \n" + "Actual targets with only one metric:\n" + "\tt1 — metric.test.1",
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
			Actual targets with only one metric:`, "\n\t\t\t", "\n"),
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
				t2 — metric.test.1`, "\n\t\t\t", "\n"),
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
