package vmselect

import (
	"fmt"
	"testing"
	"time"
)

func TestFetch(t *testing.T) {
	now := time.Now().Unix()

	from := now - 120
	until := now

	// target := `sum(rate(nginx_ingress_controller_requests{namespace=~"moira-alert"}[10m]))` //nolint
	target := `label_keep(
	rate(
		container_cpu_cfs_throttled_periods_total{
			container=~"filter",
			namespace=~"moira-alert"
		}[10m]
	) / rate(
		container_cpu_cfs_periods_total{
			container=~"filter",
			namespace=~"moira-alert"
		}[10m]
	) * 100 , "pod")`

	remote := Create(&Config{})
	res, err := remote.Fetch(target, from, until, false)
	if err != nil {
		fmt.Printf("error: %s\n\n", err.Error())
		t.Fail()
	}

	fmt.Printf("%+v\n\n", res)
}
