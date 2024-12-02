package main

import (
	"context"
	"testing"
)

func TestReadWriteMetric(t *testing.T) {
	_, database := makeDb()
	// finish := make(chan struct{})
	// readWriteMetrics(database, logger, finish, 0)
	ctx := context.Background()
	client := database.Client()
	client.Ping(ctx)
}
