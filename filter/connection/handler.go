package connection

import (
	"bufio"
	"io"
	"net"
	"sync"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/filter"
)

// Handler handling connection data and shift it to MatchedMetrics channel
type Handler struct {
	logger          moira.Logger
	patternsStorage *filter.PatternStorage
	wg              sync.WaitGroup
	terminate       chan bool
}

// NewConnectionsHandler creates new Handler
func NewConnectionsHandler(logger moira.Logger, patternsStorage *filter.PatternStorage) *Handler {
	return &Handler{
		logger:          logger,
		patternsStorage: patternsStorage,
		terminate:       make(chan bool, 1),
	}
}

// HandleConnection convert every line from connection to metric and send it to MatchedMetric channel
func (handler *Handler) HandleConnection(connection net.Conn, matchedMetricsChan chan *moira.MatchedMetric) {
	handler.wg.Add(1)
	go func() {
		defer handler.wg.Done()
		handler.handle(connection, matchedMetricsChan)
	}()
}

func (handler *Handler) handle(connection net.Conn, matchedMetricsChan chan *moira.MatchedMetric) {
	buffer := bufio.NewReader(connection)

	go func(conn net.Conn) {
		<-handler.terminate
		conn.Close()
	}(connection)

	for {
		lineBytes, err := buffer.ReadBytes('\n')
		if err != nil {
			connection.Close()
			if err != io.EOF {
				handler.logger.Errorf("read failed: %s", err)
			}
			break
		}
		lineBytes = lineBytes[:len(lineBytes)-1]
		handler.wg.Add(1)
		go func(ch chan *moira.MatchedMetric) {
			defer handler.wg.Done()
			if m := handler.patternsStorage.ProcessIncomingMetric(lineBytes); m != nil {
				ch <- m
			}
		}(matchedMetricsChan)
	}
}

// StopHandlingConnections closes all open connections and wait for handling ramaining metrics
func (handler *Handler) StopHandlingConnections() {
	close(handler.terminate)
	handler.wg.Wait()
}
