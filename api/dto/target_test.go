package dto

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-graphite/carbonapi/pkg/parser"
	"github.com/moira-alert/moira"
	"github.com/stretchr/testify/require"
)

func TestTargetVerification(t *testing.T) {
	t.Run("Target verification", func(t *testing.T) {
		t.Run("Check unknown trigger type", func(t *testing.T) {
			targets := []string{`alias(test.one,'One'`}
			problems, err := TargetVerification(targets, 10, "random_source")
			require.Equal(t, fmt.Errorf("unknown trigger source '%s'", "random_source"), err)
			require.Nil(t, problems)
		})

		t.Run("Check bad function", func(t *testing.T) {
			targets := []string{`alias(test.one,'One'`}
			problems, err := TargetVerification(targets, 10, moira.GraphiteLocal)
			require.NoError(t, err)
			require.Len(t, problems, 1)
			require.False(t, problems[0].SyntaxOk)
		})

		t.Run("Check target with unrecognized syntax error", func(t *testing.T) {
			targets := []string{`alias(test.one,'One'))`}
			problems, err := TargetVerification(targets, 10, moira.GraphiteLocal)
			require.NoError(t, err)
			require.False(t, problems[0].SyntaxOk)
			require.Nil(t, problems[0].TreeOfProblems)
		})

		t.Run("Check correct construction", func(t *testing.T) {
			targets := []string{`alias(test.one,'One')`}
			problems, err := TargetVerification(targets, 10, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
		})

		t.Run("Check correct empty function", func(t *testing.T) {
			targets := []string{`alias(movingSum(),'One')`}
			problems, err := TargetVerification(targets, 10, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Nil(t, problems[0].TreeOfProblems)
		})

		t.Run("Check interval larger that TTL", func(t *testing.T) {
			targets := []string{"movingAverage(groupByTags(seriesByTag('project=my-test-project'), 'max'), '10min')"}
			problems, err := TargetVerification(targets, 5*time.Minute, moira.GraphiteLocal)
			require.NoError(t, err)
			// target is not valid because set of metrics by last 5 minutes is not enough for function with 10min interval
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "movingAverage", problems[0].TreeOfProblems.Argument)
		})

		// potentially unreal case, because we have TTL > 0 in configs
		t.Run("Check ttl is 0", func(t *testing.T) {
			targets := []string{"movingAverage(groupByTags(seriesByTag('project=my-test-project'), 'max'), '10min')"}
			// ttl is 0 means that metrics will persist forever
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			// target is valid because there is enough metrics
			require.True(t, problems[0].SyntaxOk)
			require.Nil(t, problems[0].TreeOfProblems)
		})

		t.Run("Check unstable function", func(t *testing.T) {
			targets := []string{"summarize(test.metric, '10min')"}
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "summarize", problems[0].TreeOfProblems.Argument)
		})

		t.Run("Check false notifications function", func(t *testing.T) {
			targets := []string{"highest(test.metric)"}
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "highest", problems[0].TreeOfProblems.Argument)
		})

		t.Run("Check visual function", func(t *testing.T) {
			targets := []string{"consolidateBy(Servers.web01.sda1.free_space, 'max')"}
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "consolidateBy", problems[0].TreeOfProblems.Argument)
		})

		t.Run("Check unsupported function", func(t *testing.T) {
			targets := []string{"myUnsupportedFunction(Servers.web01.sda1.free_space, 'max')"}
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "myUnsupportedFunction", problems[0].TreeOfProblems.Argument)
		})

		t.Run("Check nested function", func(t *testing.T) {
			targets := []string{"movingAverage(myUnsupportedFunction(), '10min')"}
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "myUnsupportedFunction", problems[0].TreeOfProblems.Problems[0].Argument)
		})

		t.Run("Check target only with metric (without Graphite-function)", func(t *testing.T) {
			targets := []string{"my.metric"}
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Nil(t, problems[0].TreeOfProblems)
		})

		t.Run("Check target with space symbol in metric name", func(t *testing.T) {
			targets := []string{"a b"}
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.False(t, problems[0].SyntaxOk)
			require.Nil(t, problems[0].TreeOfProblems)
		})

		t.Run("Check seriesByTag target without non-regex args, regex has anchors", func(t *testing.T) {
			targets := []string{"seriesByTag('name=~^tag\\..*$')"}
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "seriesByTag('name=~^tag\\..*$')", problems[0].TreeOfProblems.Argument)
			require.Equal(t, isBad, problems[0].TreeOfProblems.Type)
		})

		t.Run("Check seriesByTag target with a wildcard argument", func(t *testing.T) {
			targets := []string{"seriesByTag('name=ab.bc.*.cd.*.ef')"}
			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "seriesByTag('name=ab.bc.*.cd.*.ef')", problems[0].TreeOfProblems.Argument)
			require.Equal(t, isBad, problems[0].TreeOfProblems.Type)
		})

		t.Run("Check nested seriesByTag target without non-regex args", func(t *testing.T) {
			targets := []string{"aliasByTags(seriesByTag('name=~*'),'tag')"}

			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "seriesByTag('name=~*')", problems[0].TreeOfProblems.Problems[0].Argument)
			require.Equal(t, isBad, problems[0].TreeOfProblems.Problems[0].Type)
		})

		t.Run("Check nested seriesByTag target without arguments that have strict equality", func(t *testing.T) {
			targets := []string{"aliasByTags(seriesByTag('name=~*', 'tag1~=*val1*', 'tag2=*val2*'),'tag')"}

			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Equal(t, "seriesByTag('name=~*', 'tag1~=*val1*', 'tag2=*val2*')", problems[0].TreeOfProblems.Problems[0].Argument)
			require.Equal(t, isBad, problems[0].TreeOfProblems.Problems[0].Type)
		})

		t.Run("Check nested seriesByTag target with an argument that has strict equality", func(t *testing.T) {
			targets := []string{"aliasByTags(seriesByTag('name=~*', 'tag1=val1', 'tag2=val2', 'tag3=val3*'),'tag')"}

			problems, err := TargetVerification(targets, 0, moira.GraphiteLocal)
			require.NoError(t, err)
			require.True(t, problems[0].SyntaxOk)
			require.Nil(t, problems[0].TreeOfProblems)
		})
	})
}

func TestConvertGraphiteTimeToTimeDuration(t *testing.T) {
	t.Run("Test graphite time functions", func(t *testing.T) {
		for _, data := range getTestDataTargetWithTimeInterval() {
			expr, _, err := parser.ParseExpr(data.target)
			require.NoError(t, err)

			if len(expr.Args()) < 2 {
				continue
			}

			_, expected := positiveDuration(expr.Args()[1])
			require.Equal(t, expected, data.actual)
		}
	})
}

func TestParseParametersToTimeDuration(t *testing.T) {
	tmpl := "timeShift(Sales.widgets.largeBlue,'%s')"
	tmplInt := "timeShift(Sales.widgets.largeBlue, %d)"

	t.Run("Strings", func(t *testing.T) {
		var expr parser.Expr

		var err error

		for tmplTime, actual := range getTimes() {
			switch value := tmplTime.(type) {
			case int:
				expr, _, err = parser.ParseExpr(fmt.Sprintf(tmplInt, value))
			case string:
				expr, _, err = parser.ParseExpr(fmt.Sprintf(tmpl, value))
			}

			require.NoError(t, err)

			_, expected := positiveDuration(expr.Args()[1])
			require.Equal(t, expected, actual)
		}
	})
}

func TestFuncIsSupported(t *testing.T) {
	t.Run("Test supported functions", func(t *testing.T) {
		t.Run("func supported", func(t *testing.T) {
			ok := funcIsSupported("divideSeries")
			require.True(t, ok)

			ok = funcIsSupported("absolute")
			require.True(t, ok)

			ok = funcIsSupported("alias")
			require.True(t, ok)

			ok = funcIsSupported("aliasByMetric")
			require.True(t, ok)

			ok = funcIsSupported("aliasByNode")
			require.True(t, ok)
		})

		t.Run("func not supported", func(t *testing.T) {
			ok := funcIsSupported("IAmNotSupported")
			require.False(t, ok)
		})
	})
}

func TestDoesAnyTreeHaveError(t *testing.T) {
	type testCase struct {
		name  string
		trees []TreeOfProblems
		want  bool
	}

	tests := []testCase{
		{
			name: "All trees ok",
			trees: []TreeOfProblems{
				{SyntaxOk: true, TreeOfProblems: nil},
				{SyntaxOk: true, TreeOfProblems: nil},
			},
			want: false,
		},
		{
			name: "One tree syntax error",
			trees: []TreeOfProblems{
				{SyntaxOk: false, TreeOfProblems: nil},
				{SyntaxOk: true, TreeOfProblems: nil},
			},
			want: true,
		},
		{
			name: "One tree has error",
			trees: []TreeOfProblems{
				{SyntaxOk: true, TreeOfProblems: &ProblemOfTarget{Type: isBad}},
				{SyntaxOk: true, TreeOfProblems: nil},
			},
			want: true,
		},
		{
			name:  "Empty slice",
			trees: []TreeOfProblems{},
			want:  false,
		},
		{
			name: "Multiple trees, mixed errors",
			trees: []TreeOfProblems{
				{SyntaxOk: true, TreeOfProblems: nil},
				{SyntaxOk: false, TreeOfProblems: nil},
				{SyntaxOk: true, TreeOfProblems: &ProblemOfTarget{Type: isBad}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DoesAnyTreeHaveError(tt.trees)
			assert.Equal(t, tt.want, got, "DoesAnyTreeHaveError() for %s", tt.name)
		})
	}
}

func getTestDataTargetWithTimeInterval() []struct {
	target string
	actual time.Duration
} {
	return []struct {
		target string
		actual time.Duration
	}{
		{"linearRegression(Server.instance*.threads.busy, \"00:00 20140101\",\"11:59 20140630\")", 0},
		{"divideSeries(server.FreeSpace,delay(server.FreeSpace,1))", 0},
		{"timeSlice(network.core.port1,\"00:00 20140101\",\"11:59 20140630\")", 0},
		{"timeSlice(network.core.port1,\"12:00 20140630\",\"now\")", 0},
		{"delay(server.FreeSpace,1)", time.Second},
		{"exponentialMovingAverage(*.transactions.count, 10)", time.Second * 10},
		{"exponentialMovingAverage(*.transactions.count, '-10s')", time.Second * 10},
		{"integralByInterval(company.sales.perMinute, \"1d\")&from=midnight-10days", time.Hour * 24},
		{"linearRegression(Server.instance01.threads.busy, '-1d')", time.Hour * 24},
		{"movingAverage(Server.instance*.threads.idle,'5min')", time.Minute * 5},
		{"movingAverage(Server.instance01.threads.busy,10)", time.Second * 10},
		{"movingMax(Server.instance01.requests,10)", time.Second * 10},
		{"movingMax(Server.instance*.errors,'5min')", time.Minute * 5},
		{"movingMedian(Server.instance01.threads.busy,10)", time.Second * 10},
		{"movingMedian(Server.instance*.threads.idle,'5min')", time.Minute * 5},
		{"movingMin(Server.instance01.requests,10)", time.Second * 10},
		{"movingMin(Server.instance*.errors,'5min')", time.Minute * 5},
		{"movingSum(Server.instance01.requests,10)", time.Second * 10},
		{"movingSum(Server.instance*.errors,'5min')", time.Minute * 5},
		{"movingWindow(Server.instance01.threads.busy,10)", time.Second * 10},
		{"movingWindow(Server.instance*.threads.idle,'5min','median',0.5)", time.Minute * 5},
		{"randomWalk(\"The.time.series\", 60)", time.Second * 60},
		{"sin(\"The.time.series\", 2)", time.Second * 2},
		{"summarize(counter.errors, \"1hour\")", time.Hour},
		{"summarize(nonNegativeDerivative(gauge.num_users), \"1week\")", time.Hour * 24 * 7},
		{"summarize(queue.size, \"1hour\", \"avg\")", time.Hour},
		{"summarize(queue.size, \"1hour\", \"max\")", time.Hour},
		{"summarize(metric, \"13week\", \"avg\", true)", time.Hour * 24 * 7 * 13},
		{"time(\"The.time.series\")", 0},
		{"time(\"The.time.series\", 120)", time.Minute * 2},
		{"timeShift(Sales.widgets.largeBlue,\"7d\")", time.Hour * 24 * 7},
		{"timeShift(Sales.widgets.largeBlue,\"-7d\")", time.Hour * 24 * 7},
		{"timeShift(Sales.widgets.largeBlue,\"+1h\")", time.Hour},
		{"timeStack(Sales.widgets.largeBlue,\"1d\",0,7)", time.Hour * 24},
	}
}

func getTimes() map[any]time.Duration {
	return map[any]time.Duration{
		// Integer
		1:  time.Second,
		2:  time.Second * 2,
		10: time.Second * 10,
		60: time.Second * 60,
		// Strings
		"-10s":   time.Second * 10,
		"5min":   time.Minute * 5,
		"-1h":    time.Hour,
		"1hour":  time.Hour,
		"1d":     time.Hour * 24,
		"-1d":    time.Hour * 24,
		"1week":  time.Hour * 24 * 7,
		"-1week": time.Hour * 24 * 7,
		"13week": time.Hour * 24 * 7 * 13,
		// Bad
		"now": 0, "00:00 20140101": 0, "11:59 20140630": 0,
	}
}
