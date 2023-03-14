package local

import (
	"github.com/go-graphite/carbonapi/expr/functions"
	"github.com/go-graphite/carbonapi/expr/rewrite"
	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

// Local is implementation of MetricSource interface, which implements fetch metrics method from moira database installation
type Local struct {
	database moira.Database
}

// Create configures local metric source
func Create(dataBase moira.Database) metricSource.MetricSource {
	// configure carbon-api functions
	rewrite.New(make(map[string]string))
	functions.New(make(map[string]string))

	return &Local{
		database: dataBase,
	}
}

// GetMetricsTTLSeconds returns metrics lifetime in Redis
func (local *Local) GetMetricsTTLSeconds() int64 {
	return local.database.GetMetricsTTLSeconds()
}

// IsConfigured always returns true. It easy to configure local source =)
func (local *Local) IsConfigured() (bool, error) {
	return true, nil
}

// Fetch is analogue of evaluateTarget method in graphite-web, that gets target metrics value from DB and Evaluate it using carbon-api eval package
func (local *Local) Fetch(target string, from int64, until int64, allowRealTimeAlerting bool) (metricSource.FetchResult, error) {
	// Don't fetch intervals larger than metrics TTL to prevent OOM errors
	// See https://github.com/moira-alert/moira/pull/519
	from = moira.MaxInt64(from, until-local.database.GetMetricsTTLSeconds())

	result := CreateEmptyFetchResult()
	ctx := evalCtx{from, until}

	err := ctx.fetchAndEval(local.database, target, result)
	if err != nil {
		return nil, err
	}

	if allowRealTimeAlerting {
		return result, nil
	}

	for i := range result.MetricsData {
		metricData := &result.MetricsData[i]
		if len(metricData.Values) == 0 {
			continue
		}
		metricData.Values = metricData.Values[:len(metricData.Values)-1]
	}

	return result, nil
}
