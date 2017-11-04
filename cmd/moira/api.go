package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/handler"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/logging/go-logging"
)

// APIService is a HTTP server for Moira API
type APIService struct {
	Config         *api.Config
	DatabaseConfig *redis.Config

	LogFile  string
	LogLevel string

	http *http.Server
}

// Start Moira API HTTP server
func (apiService *APIService) Start() error {
	logger, err := logging.ConfigureLog(apiService.LogFile, apiService.LogLevel, "api")
	if err != nil {
		return fmt.Errorf("Can't configure logger for Api: %v", err)
	}

	if !apiService.Config.Enabled {
		logger.Info("Moira Api Disabled")
		return nil
	}

	dataBase := redis.NewDatabase(logger, *apiService.DatabaseConfig)
	listener, err := net.Listen("tcp", apiService.Config.Listen)
	if err != nil {
		return err
	}

	httpHandler := handler.NewHandler(dataBase, logger, apiService.Config)
	apiService.http = &http.Server{
		Handler: httpHandler,
	}

	go func() {
		apiService.http.Serve(listener)
	}()

	logger.Info("Moira Api Started")
	return nil
}

// Stop Moira API HTTP server
func (apiService *APIService) Stop() error {
	if !apiService.Config.Enabled {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return apiService.http.Shutdown(ctx)
}
