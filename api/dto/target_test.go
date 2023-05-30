package dto

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-graphite/carbonapi/pkg/parser"
	"github.com/moira-alert/moira"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTargetVerification(t *testing.T) {
	Convey("Target verification", t, func() {
		Convey("Check bad function", func() {
			targets := []string{`alias(test.one,'One'`}
			problems, _ := TargetVerification(targets, 10, moira.GraphiteLocal)
			So(len(problems), ShouldEqual, 1)
			So(problems[0].SyntaxOk, ShouldBeFalse)
		})

		Convey("Check correct construction", func() {
			targets := []string{`alias(test.one,'One')`}
			problems, _ := TargetVerification(targets, 10, moira.GraphiteLocal)
			So(problems[0].SyntaxOk, ShouldBeTrue)
		})

		Convey("Check correct empty function", func() {
			targets := []string{`alias(movingSum(),'One')`}
			problems, _ := TargetVerification(targets, 10, moira.GraphiteLocal)
			So(problems[0].SyntaxOk, ShouldBeTrue)
			So(problems[0].TreeOfProblems, ShouldBeNil)
		})

		Convey("Check interval larger that TTL", func() {
			targets := []string{"movingAverage(groupByTags(seriesByTag('project=my-test-project'), 'max'), '10min')"}
			problems, _ := TargetVerification(targets, 5*time.Minute, moira.GraphiteLocal)
			// target is not valid because set of metrics by last 5 minutes is not enough for function with 10min interval
			So(problems[0].SyntaxOk, ShouldBeTrue)
			So(problems[0].TreeOfProblems.Argument, ShouldEqual, "movingAverage")
		})

		// potentially unreal case, because we have TTL > 0 in configs
		Convey("Check ttl is 0", func() {
			targets := []string{"movingAverage(groupByTags(seriesByTag('project=my-test-project'), 'max'), '10min')"}
			// ttl is 0 means that metrics will persist forever
			problems, _ := TargetVerification(targets, 0, moira.GraphiteLocal)
			// target is valid because there is enough metrics
			So(problems[0].SyntaxOk, ShouldBeTrue)
			So(problems[0].TreeOfProblems, ShouldBeNil)
		})

		Convey("Check unstable function", func() {
			targets := []string{"summarize(test.metric, '10min')"}
			problems, _ := TargetVerification(targets, 0, moira.GraphiteLocal)
			So(problems[0].SyntaxOk, ShouldBeTrue)
			So(problems[0].TreeOfProblems.Argument, ShouldEqual, "summarize")
		})

		Convey("Check false notifications function", func() {
			targets := []string{"highest(test.metric)"}
			problems, _ := TargetVerification(targets, 0, moira.GraphiteLocal)
			So(problems[0].SyntaxOk, ShouldBeTrue)
			So(problems[0].TreeOfProblems.Argument, ShouldEqual, "highest")
		})

		Convey("Check visual function", func() {
			targets := []string{"consolidateBy(Servers.web01.sda1.free_space, 'max')"}
			problems, _ := TargetVerification(targets, 0, moira.GraphiteLocal)
			So(problems[0].SyntaxOk, ShouldBeTrue)
			So(problems[0].TreeOfProblems.Argument, ShouldEqual, "consolidateBy")
		})

		Convey("Check unsupported function", func() {
			targets := []string{"myUnsupportedFunction(Servers.web01.sda1.free_space, 'max')"}
			problems, _ := TargetVerification(targets, 0, moira.GraphiteLocal)
			So(problems[0].SyntaxOk, ShouldBeTrue)
			So(problems[0].TreeOfProblems.Argument, ShouldEqual, "myUnsupportedFunction")
		})

		Convey("Check nested function", func() {
			targets := []string{"movingAverage(myUnsupportedFunction(), '10min')"}
			problems, _ := TargetVerification(targets, 0, moira.GraphiteLocal)
			So(problems[0].SyntaxOk, ShouldBeTrue)
			So(problems[0].TreeOfProblems.Problems[0].Argument, ShouldEqual, "myUnsupportedFunction")
		})

		Convey("Check target only with metric (without Graphite-function)", func() {
			targets := []string{"my.metric"}
			problems, _ := TargetVerification(targets, 0, moira.GraphiteLocal)
			So(problems[0].SyntaxOk, ShouldBeTrue)
			So(problems[0].TreeOfProblems, ShouldBeNil)
		})

		Convey("Check target with space symbol in metric name", func() {
			targets := []string{"a b"}
			problems, _ := TargetVerification(targets, 0, moira.GraphiteLocal)
			So(problems[0].SyntaxOk, ShouldBeFalse)
			So(problems[0].TreeOfProblems, ShouldBeNil)
		})
	})
}

func TestConvertGraphiteTimeToTimeDuration(t *testing.T) {
	Convey("Test graphite time functions", t, func() {
		for _, data := range getTestDataTargetWithTimeInterval() {
			expr, _, err := parser.ParseExpr(data.target)
			So(err, ShouldBeNil)

			if len(expr.Args()) < 2 {
				continue
			}
			_, expected := positiveDuration(expr.Args()[1])
			So(expected, ShouldEqual, data.actual)
		}
	})
}

func TestParseParametersToTimeDuration(t *testing.T) {
	tmpl := "timeShift(Sales.widgets.largeBlue,'%s')"
	tmplInt := "timeShift(Sales.widgets.largeBlue, %d)"

	Convey("Strings", t, func() {
		var expr parser.Expr
		var err error

		for tmplTime, actual := range getTimes() {
			switch value := tmplTime.(type) {
			case int:
				expr, _, err = parser.ParseExpr(fmt.Sprintf(tmplInt, value))
			case string:
				expr, _, err = parser.ParseExpr(fmt.Sprintf(tmpl, value))
			}

			So(err, ShouldBeNil)

			_, expected := positiveDuration(expr.Args()[1])
			So(expected, ShouldEqual, actual)
		}
	})
}

func TestFuncIsSupported(t *testing.T) {
	Convey("Test supported functions", t, func() {
		Convey("func supported", func() {
			ok := funcIsSupported("divideSeries")
			So(ok, ShouldBeTrue)

			ok = funcIsSupported("absolute")
			So(ok, ShouldBeTrue)

			ok = funcIsSupported("alias")
			So(ok, ShouldBeTrue)

			ok = funcIsSupported("aliasByMetric")
			So(ok, ShouldBeTrue)

			ok = funcIsSupported("aliasByNode")
			So(ok, ShouldBeTrue)
		})

		Convey("func not supported", func() {
			ok := funcIsSupported("IAmNotSupported")
			So(ok, ShouldBeFalse)
		})
	})
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

func getTimes() map[interface{}]time.Duration {
	return map[interface{}]time.Duration{
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
