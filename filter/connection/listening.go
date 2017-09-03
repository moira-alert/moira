package connection

import (
	"fmt"
	"net"
	"sync"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/filter"
)

// MetricsListener is facade for standard net.MetricsListener and accept connection for handling it
type MetricsListener struct {
	listener net.Listener
	handler  *Handler
	logger   moira.Logger
	tomb     tomb.Tomb
}

// NewListener creates new listener
func NewListener(port string, logger moira.Logger, patternStorage *filter.PatternStorage) (*MetricsListener, error) {
	listen := port
	newListener, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, fmt.Errorf("Failed to listen on [%s]: %s", listen, err.Error())
	}
	listener := MetricsListener{
		listener: newListener,
		logger:   logger,
		handler:  NewConnectionHandler(logger, patternStorage),
	}
	return &listener, nil
}

// Listen waits for new data in connection and handles it in ConnectionHandler
// All handled data sets to metricsChan
func (listener *MetricsListener) Listen() chan *moira.MatchedMetric {
	metricsChan := make(chan *moira.MatchedMetric, 10)
	listener.tomb.Go(func() error {
		var handlerWG sync.WaitGroup
		for {
			select {
			case <-listener.tomb.Dying():
				{
					listener.logger.Info("Listener stopped")
					handlerWG.Wait()
					close(metricsChan)
					return nil
				}
			default:
				{
					conn, err := listener.listener.Accept()
					if err != nil {
						listener.logger.Infof("Failed to accept connection: %s", err.Error())
						continue
					}
					handlerWG.Add(1)
					go listener.handler.HandleConnection(conn, metricsChan, &handlerWG)
				}
			}
		}
	})
	listener.logger.Info("Listener started")
	return metricsChan
}

// Stop stops listening connection
func (listener *MetricsListener) Stop() error {
	listener.tomb.Kill(nil)
	return listener.tomb.Wait()
}
