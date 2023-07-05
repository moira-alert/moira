package vmselect

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/moira-alert/moira"
	metricSource "github.com/moira-alert/moira/metric_source"
)

type Config struct {
	Enabled       bool
	CheckInterval time.Duration
	MetricsTTL    time.Duration
	URL           string
	User          string
	Password      string
}

func Create(config *Config) metricSource.MetricSource {
	return &VMSelect{
		config: config,
		client: &http.Client{Timeout: time.Second * 60},
	}
}

type VMSelect struct {
	config *Config
	client *http.Client
}

func (vmselect *VMSelect) Fetch(target string, from int64, until int64, allowRealTimeAlerting bool) (metricSource.FetchResult, error) {
	from = moira.MaxInt64(from, until-int64(vmselect.config.MetricsTTL.Seconds()))

	r, err := vmselect.prepareRequest(from, until, target)
	if err != nil {
		return nil, err
	}

	data, err := vmselect.makeRequest(r)
	if err != nil {
		return nil, err
	}

	result := VMSelectResponce{}
	err = json.Unmarshal(data, &result)

	if err != nil {
		return nil, err
	}

	return result.ConvertToFetchResult(), nil
}

func (vmselect *VMSelect) GetMetricsTTLSeconds() int64 {
	return int64(vmselect.config.MetricsTTL.Seconds())
}

func (vmselect *VMSelect) IsConfigured() (bool, error) {
	// TODO: check if configuration is valid
	return vmselect.config.Enabled, nil
}

func (*VMSelect) IsAvailable() (bool, error) {
	// TODO: check if prometheus is actually available
	return true, nil
}

func (vmselect *VMSelect) prepareRequest(from, until int64, target string) (*http.Request, error) {
	req, err := http.NewRequest("GET", vmselect.config.URL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("query", target)
	q.Add("start", strconv.FormatInt(from, 10))
	q.Add("end", strconv.FormatInt(until, 10))
	q.Add("step", "60")
	q.Add("nocache", "1")
	req.URL.RawQuery = q.Encode()

	if vmselect.config.User != "" && vmselect.config.Password != "" {
		req.SetBasicAuth(vmselect.config.User, vmselect.config.Password)
	}
	return req, nil
}

func (vmselect *VMSelect) makeRequest(req *http.Request) ([]byte, error) {
	var body []byte

	resp, err := vmselect.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return body, fmt.Errorf("the remote server is not available or the response was reset by timeout. "+
			"TTL: %s, PATH: %s, ERROR: %v ", vmselect.client.Timeout.String(), req.URL.RawPath, err)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return body, err
	}

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("bad response status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

type VMSelectResponce struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values []VMSelectValue   `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func (resp *VMSelectResponce) ConvertToFetchResult() *FetchResult {
	result := FetchResult{
		MetricsData: make([]metricSource.MetricData, 0, len(resp.Data.Result)),
	}

	for _, res := range resp.Data.Result {
		values := []float64{}
		for _, v := range res.Values {
			values = append(values, v.Value)
		}

		data := metricSource.MetricData{
			Name:      targetFromTags(res.Metric),
			StartTime: res.Values[0].Timestamp,
			StopTime:  res.Values[len(res.Values)-1].Timestamp,
			StepTime:  60,
			Values:    values,
			Wildcard:  false,
		}
		result.MetricsData = append(result.MetricsData, data)
	}

	return &result
}

func targetFromTags(tags map[string]string) string {
	if len(tags) == 1 {
		for _, value := range tags {
			return value
		}
	}

	target := ""
	if name, ok := tags["__name__"]; ok {
		target += name
	}

	tagsList := make([]struct{ key, value string }, 0)
	for key, value := range tags {
		tagsList = append(tagsList, struct{ key, value string }{key, value})
	}

	sort.Slice(tagsList, func(i, j int) bool {
		a, b := tagsList[i], tagsList[j]
		if a.key != b.key {
			return a.key < b.key
		}

		return a.value < b.value
	})

	for _, tag := range tagsList {
		key, value := tag.key, tag.value

		if key == "__name__" {
			continue
		}
		if target != "" {
			target += ";"
		}
		target += key
		target += "="
		target += value
	}
	return target
}

type VMSelectValue struct {
	Timestamp int64
	Value     float64
}

func (b *VMSelectValue) UnmarshalJSON(data []byte) error {
	var v []interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	time, ok := v[0].(float64)
	if !ok {
		return fmt.Errorf("expected timestamp to be an number but it was %T", v[0])
	}
	b.Timestamp = int64(time)

	value, err := strconv.ParseFloat(v[1].(string), 64)
	if err != nil {
		return fmt.Errorf("error parsing the value: %w", err)
	}
	b.Value = value

	return nil
}

type FetchResult struct {
	MetricsData []metricSource.MetricData
}

// GetMetricsData return all metrics data from fetch result
func (fetchResult *FetchResult) GetMetricsData() []metricSource.MetricData {
	return fetchResult.MetricsData
}

// GetPatterns always returns error, because we can't fetch target patterns from remote metrics source
func (*FetchResult) GetPatterns() ([]string, error) {
	return make([]string, 0), fmt.Errorf("remote fetch result never returns patterns")
}

// GetPatternMetrics always returns error, because remote fetch doesn't return base pattern metrics
func (*FetchResult) GetPatternMetrics() ([]string, error) {
	return make([]string, 0), fmt.Errorf("remote fetch result never returns pattern metrics")
}
