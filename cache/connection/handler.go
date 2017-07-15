package connection

import (
	"bufio"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cache"
	"io"
	"log"
	"net"
	"sync"
)

type ConnectionHandler struct {
	logger          moira.Logger
	patternsStorage *cache.PatternStorage
}

func NewConnectionHandler(logger moira.Logger, patternsStorage *cache.PatternStorage) *ConnectionHandler {
	return &ConnectionHandler{
		logger:          logger,
		patternsStorage: patternsStorage,
	}
}

func (handler *ConnectionHandler) HandleConnection(connection net.Conn, matchedMetricsChan chan *moira.MatchedMetric, shutdown chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	connectionBuffer := bufio.NewReader(connection)
	go handler.closeConnection(connection, shutdown)

	for {
		lineBytes, err := connectionBuffer.ReadBytes('\n')
		if err != nil {
			connection.Close()
			if err != io.EOF {
				log.Printf("read failed: %s", err)
			}
			break
		}
		lineBytes = lineBytes[:len(lineBytes)-1]
		wg.Add(1)
		go handler.handleLine(lineBytes, matchedMetricsChan, wg)
	}
}

func (handler *ConnectionHandler) handleLine(lineBytes []byte, matchedMetricsChan chan *moira.MatchedMetric, wg *sync.WaitGroup) {
	defer wg.Done()
	if m := handler.patternsStorage.ProcessIncomingMetric(lineBytes); m != nil {
		matchedMetricsChan <- m
	}
}

func (handler *ConnectionHandler) closeConnection(connection net.Conn, shutdown chan bool) {
	<-shutdown
	connection.Close()
}
