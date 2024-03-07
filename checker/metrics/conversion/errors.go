package conversion

import (
	"fmt"
	"sort"
	"strings"

	metricsource "github.com/moira-alert/moira/metric_source"
)

type errUnexpectedAloneMetricBuilder struct {
	result      ErrUnexpectedAloneMetric
	returnError bool
}

func newErrUnexpectedAloneMetricBuilder() errUnexpectedAloneMetricBuilder {
	return errUnexpectedAloneMetricBuilder{
		result:      ErrUnexpectedAloneMetric{},
		returnError: false,
	}
}

func (b *errUnexpectedAloneMetricBuilder) setDeclared(declared map[string]bool) {
	b.result.declared = declared
}

func (b *errUnexpectedAloneMetricBuilder) addUnexpected(targetName string, unexpected map[string]metricsource.MetricData) {
	b.returnError = true
	if b.result.unexpected == nil {
		b.result.unexpected = make(map[string][]string)
	}
	metricNames := []string{}
	for metricName := range unexpected {
		metricNames = append(metricNames, metricName)
	}
	b.result.unexpected[targetName] = metricNames
}

func (b *errUnexpectedAloneMetricBuilder) build() error {
	if b.returnError {
		return b.result
	}
	return nil
}

// ErrUnexpectedAloneMetric is an error that fired by checker if alone metrics do not.
// match alone metrics specified in trigger.
type ErrUnexpectedAloneMetric struct {
	declared   map[string]bool
	unexpected map[string][]string
}

// Error is a function that implements error interface.
func (err ErrUnexpectedAloneMetric) Error() string {
	var builder strings.Builder

	builder.WriteString("Unexpected to have some targets with more than only one metric.\n")
	builder.WriteString("Expected targets with only one metric:")
	expectedArray := make([]string, 0, len(err.declared))
	for targetName := range err.declared {
		expectedArray = append(expectedArray, targetName)
	}
	sort.Strings(expectedArray)
	for i, targetName := range expectedArray {
		if i > 0 {
			builder.WriteRune(',')
		}
		builder.WriteRune(' ')
		builder.WriteString(targetName)
	}
	builder.WriteRune('\n')

	builder.WriteString("Targets with multiple metrics but that declared as targets with alone metrics:")
	actualArray := make([]string, 0, len(err.unexpected))
	for targetName := range err.unexpected {
		actualArray = append(actualArray, targetName)
	}
	sort.Strings(actualArray)
	for _, targetName := range actualArray {
		builder.WriteRune('\n')
		builder.WriteRune('\t')
		builder.WriteString(targetName)
		builder.WriteString(" — ")
		builder.WriteString(strings.Join(err.unexpected[targetName], ", "))
	}
	return builder.String()
}

// NewErrEmptyAloneMetricsTarget constructor function for ErrEmptyAloneMetricsTarget.
func NewErrEmptyAloneMetricsTarget(targetName string) error {
	return ErrEmptyAloneMetricsTarget{
		targetName: targetName,
	}
}

// ErrEmptyAloneMetricsTarget is an error that raise in situation when target marked as alone metrics target.
// but do not have metrics yet and do not have metrics saved in last check.
type ErrEmptyAloneMetricsTarget struct {
	targetName string
}

// Error is an error interface implementation for ErrEmptyAloneMetricsTarget.
func (e ErrEmptyAloneMetricsTarget) Error() string {
	return fmt.Sprintf("target %s declared as alone metrics target but do not have any metrics and saved state in last check", e.targetName)
}
