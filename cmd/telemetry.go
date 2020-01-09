package cmd

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/moira-alert/moira"
)

func StartTelemetryServer(logger moira.Logger, listen string, profilerConfig ProfilerConfig) (func(), error) {
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, err
	}
	serverMux := http.NewServeMux()
	if profilerConfig.Enabled {
		serverMux.HandleFunc("/debug/pprof/", pprof.Index)
		serverMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		serverMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		serverMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		serverMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	server := &http.Server{Handler: serverMux}
	go func() {
		server.Serve(listener)
	}()
	stopServer := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Errorf("Can't stop telemetry server correctly: %v", err)
		}
	}
	return stopServer, nil
}
