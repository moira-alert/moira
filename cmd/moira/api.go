package main

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/handler"
)

// APIServer is a HTTP server for Moira API
type APIServer struct {
	Config *api.Config
	Log    moira.Logger
	DB     moira.Database
	http   *http.Server
}

// Start Moira API HTTP server
func (server *APIServer) Start() error {
	if !server.Config.Enabled {
		server.Log.Debug("API Disabled")
		return nil
	}

	listener, err := net.Listen("tcp", server.Config.Listen)
	if err != nil {
		return err
	}

	httpHandler := handler.NewHandler(server.DB, server.Log)

	server.http = &http.Server{
		Handler: httpHandler,
	}

	go func() {
		server.http.Serve(listener)
	}()

	server.Log.Info("API Started")
	return nil
}

// Stop Moira API HTTP server
func (server *APIServer) Stop() error {
	if !server.Config.Enabled {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.http.Shutdown(ctx)
}
