package prometheus

import (
	"encoding/base64"
	"fmt"

	"github.com/prometheus/client_golang/api"
	prometheusApi "github.com/prometheus/client_golang/api/prometheus/v1"
	promConfig "github.com/prometheus/common/config"
)

func createPrometheusApi(config *Config) (prometheusApi.API, error) {
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

	return prometheusApi.NewAPI(promCl), nil
}
