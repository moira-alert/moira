package connection

import (
	"bufio"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cache"
	"io"
	"net"
	"sync"
)

//Handler handling connection data and shift it to MatchedMetrics channel
type Handler struct {
	logger          moira.Logger
	patternsStorage *cache.PatternStorage
}

//NewConnectionHandler creates new Handler
func NewConnectionHandler(logger moira.Logger, patternsStorage *cache.PatternStorage) *Handler {
	return &Handler{
		logger:          logger,
		patternsStorage: patternsStorage,
	}
}

//HandleConnection convert every line from connection to metric and send it to MatchedMetric channel
func (handler *Handler) HandleConnection(connection net.Conn, matchedMetricsChan chan *moira.MatchedMetric, shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	connectionBuffer := bufio.NewReader(connection)
	go handler.closeConnection(connection, shutdown)

	for {
		lineBytes, err := connectionBuffer.ReadBytes('\n')
		if err != nil {
			connection.Close()
			if err != io.EOF {
				handler.logger.Errorf("read failed: %s", err)
			}
			break
		}
		lineBytes = lineBytes[:len(lineBytes)-1]
		wg.Add(1)
		go handler.handleLine(lineBytes, matchedMetricsChan, wg)
	}
}

func (handler *Handler) handleLine(lineBytes []byte, matchedMetricsChan chan *moira.MatchedMetric, wg *sync.WaitGroup) {
	defer wg.Done()
	if m := handler.patternsStorage.ProcessIncomingMetric(lineBytes); m != nil {
		matchedMetricsChan <- m
	}
}

func (handler *Handler) closeConnection(connection net.Conn, shutdown chan bool) {
	<-shutdown
	connection.Close()
}
