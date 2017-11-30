package cmd

import (
	"github.com/moira-alert/moira"
	"net/http"
	_ "net/http/pprof"
)

// StartProfiling starts http server with profiling data at given port
func StartProfiling(logger moira.Logger, config ProfilerConfig) {

	go func() {
		err := http.ListenAndServe(config.Port, nil)
		if err != nil {
			logger.Infof("Can't start pprof server: %v", err)
			return
		}
	}()
}
