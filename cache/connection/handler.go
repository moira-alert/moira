package connection

import (
	"bufio"
	"io"
	"net"
	"sync"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cache"
)

// Handler handling connection data and shift it to MatchedMetrics channel
type Handler struct {
	logger          moira.Logger
	patternsStorage *cache.PatternStorage
}

// NewConnectionHandler creates new Handler
func NewConnectionHandler(logger moira.Logger, patternsStorage *cache.PatternStorage) *Handler {
	return &Handler{
		logger:          logger,
		patternsStorage: patternsStorage,
	}
}

// HandleConnection convert every line from connection to metric and send it to MatchedMetric channel
func (handler *Handler) HandleConnection(connection net.Conn, matchedMetricsChan chan *moira.MatchedMetric, wg *sync.WaitGroup) {
	buffer := bufio.NewReader(connection)
	defer func() {
		connection.Close()
		wg.Done()
	}()
	var handleLineWG sync.WaitGroup
	for {
		lineBytes, err := buffer.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				handler.logger.Errorf("read failed: %s", err)
			}
			break
		}
		lineBytes = lineBytes[:len(lineBytes)-1]
		handleLineWG.Add(1)
		go handler.handleLine(lineBytes, matchedMetricsChan, &handleLineWG)
	}
	handleLineWG.Wait()
}

func (handler *Handler) handleLine(lineBytes []byte, matchedMetricsChan chan *moira.MatchedMetric, wg *sync.WaitGroup) {
	defer wg.Done()
	if m := handler.patternsStorage.ProcessIncomingMetric(lineBytes); m != nil {
		matchedMetricsChan <- m
	}
}
