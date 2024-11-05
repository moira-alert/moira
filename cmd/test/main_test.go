package main

import "testing"

func TestReadWriteMetric(t *testing.T) {
	logger, database := makeDb()
	finish := make(chan struct{})
	readWriteMetrics(database, logger, finish, 0)
}
