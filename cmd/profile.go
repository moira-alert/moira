package cmd

import (
	"net/http"

	_ "net/http/pprof"

	"github.com/moira-alert/moira"
)

func StartProfiling(logger moira.Logger, config ProfilerConfig) {

	go func() {
		err := http.ListenAndServe(config.Port, nil)
		if err != nil {
			logger.Infof("Can't start pprof server: %v", err)
			return
		}
	}()
}
