package remote

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-graphite/carbonapi/expr/types"
	"github.com/moira-alert/moira/target"

	pb "github.com/go-graphite/carbonzipper/carbonzipperpb3"
)

type graphiteMetric struct {
	Target     string
	Datapoints [][2]*float64
}

// Config represents config from remote storage
type Config struct {
	URL           string
	CheckInterval time.Duration
	Timeout       time.Duration
	User          string
	Password      string
	Enabled       bool
}

func (c *Config) IsEnabled() bool {
	return c.Enabled && c.URL != ""
}

func prepareRequest(from, until int64, target string, cfg *Config) (*http.Request, error) {
	req, err := http.NewRequest("GET", cfg.URL, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("format", "json")
	q.Add("from", strconv.FormatInt(from, 10))
	q.Add("target", target)
	q.Add("until", strconv.FormatInt(until, 10))
	req.URL.RawQuery = q.Encode()
	if cfg.User != "" && cfg.Password != "" {
		req.SetBasicAuth(cfg.User, cfg.Password)
	}
	return req, nil
}

func makeRequest(req *http.Request, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return body, err
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("bad response status %d: %s", resp.StatusCode, string(body))
		return body, err
	}
	return body, err
}

func decodeBody(body []byte) ([]*types.MetricData, error) {
	var tmp []graphiteMetric
	err := json.Unmarshal(body, &tmp)
	if err != nil {
		return nil, err
	}
	res := make([]*types.MetricData, 0, len(tmp))
	for _, m := range tmp {
		stepTime := int32(60)
		if len(m.Datapoints) > 1 {
			stepTime = int32(*m.Datapoints[1][1] - *m.Datapoints[0][1])
		}
		pbResp := pb.FetchResponse{
			Name:      m.Target,
			StartTime: int32(*m.Datapoints[0][1]),
			StopTime:  int32(*m.Datapoints[len(m.Datapoints)-1][1]),
			StepTime:  stepTime,
			Values:    make([]float64, len(m.Datapoints)),
			IsAbsent:  make([]bool, len(m.Datapoints)),
		}
		for i, v := range m.Datapoints {
			if v[0] == nil {
				pbResp.Values[i] = math.NaN()
				pbResp.IsAbsent[i] = true
			} else {
				pbResp.Values[i] = *v[0]
				pbResp.IsAbsent[i] = false
			}
		}
		res = append(res, &types.MetricData{
			FetchResponse: pbResp,
		})
	}

	return res, nil
}

func convertResponse(r []*types.MetricData, allowRealTimeAlerting bool) []*target.TimeSeries {
	ts := make([]*target.TimeSeries, len(r))
	for i, md := range r {
		if !allowRealTimeAlerting {
			last := len(md.Values) - 1
			// remove last value
			md.Values = md.Values[:last]
			md.IsAbsent = md.IsAbsent[:last]
		}
		ts[i] = &target.TimeSeries{MetricData: *md, Wildcard: false}
	}

	return ts
}

// Fetch fetches remote metrics and converts them to expected format
func Fetch(cfg *Config, target string, from, until int64, allowRealTimeAlerting bool) ([]*target.TimeSeries, error) {
	req, err := prepareRequest(from, until, target, cfg)
	if err != nil {
		return nil, err
	}
	body, err := makeRequest(req, cfg.Timeout)
	if err != nil {
		return nil, err
	}
	resp, err := decodeBody(body)
	if err != nil {
		return nil, err
	}
	return convertResponse(resp, allowRealTimeAlerting), nil
}
