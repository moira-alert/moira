package connection

import (
	"bufio"
	"io"
	"net"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/filter"
)

// Handler handling connection data and shift it to MatchedMetrics channel
type Handler struct {
	logger          moira.Logger
	patternsStorage *filter.PatternStorage
	tomb            tomb.Tomb
}

// NewConnectionHandler creates new Handler
func NewConnectionHandler(logger moira.Logger, patternsStorage *filter.PatternStorage) *Handler {
	return &Handler{
		logger:          logger,
		patternsStorage: patternsStorage,
	}
}

// HandleConnection convert every line from connection to metric and send it to MatchedMetric channel
func (handler *Handler) HandleConnection(connection net.Conn, matchedMetricsChan chan *moira.MatchedMetric) error {
	buffer := bufio.NewReader(connection)
	defer connection.Close()
	for {
		select {
		case <-handler.tomb.Dying():
			connection.Close()
			return nil
		default:
			lineBytes, err := buffer.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					handler.logger.Errorf("read failed: %s", err)
				}
				return nil
			}
			lineBytes = lineBytes[:len(lineBytes)-1]
			if m := handler.patternsStorage.ProcessIncomingMetric(lineBytes); m != nil {
				matchedMetricsChan <- m
			}
		}
	}
}
