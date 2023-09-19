package prometheus

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/prometheus/client_golang/api"
	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
	promConfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

type PrometheusApi interface {
	QueryRange(ctx context.Context, query string, r promApi.Range, opts ...promApi.Option) (model.Value, promApi.Warnings, error)
}

func createPrometheusApi(config *Config) (promApi.API, error) {
	roundTripper := api.DefaultRoundTripper

	if config.User != "" && config.Password != "" {
		rawToken := fmt.Sprintf("%s:%s", config.User, config.Password)
		token := base64.StdEncoding.EncodeToString([]byte(rawToken))

		roundTripper = promConfig.NewAuthorizationCredentialsRoundTripper(
			"Basic",
			promConfig.Secret(token),
			roundTripper,
		)
	}

	promClientConfig := api.Config{
		Address:      config.URL,
		RoundTripper: roundTripper,
	}

	promCl, err := api.NewClient(promClientConfig)
	if err != nil {
		return nil, err
	}

	return promApi.NewAPI(promCl), nil
}
