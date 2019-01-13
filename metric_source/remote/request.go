package remote

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

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
