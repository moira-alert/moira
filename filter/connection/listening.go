package connection

import (
	"fmt"
	"net"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/filter"
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
		handler:  NewConnectionsHandler(logger, patternStorage),
	}
	return &listener, nil
}

// Listen waits for new data in connection and handles it in ConnectionHandler
// All handled data sets to metricsChan
func (listener *MetricsListener) Listen() chan *moira.MatchedMetric {
	metricsChan := make(chan *moira.MatchedMetric, 10)
	listener.tomb.Go(func() error {
		for {
			select {
			case <-listener.tomb.Dying():
				{
					listener.logger.Info("Moira Filter Listener Stopped")
					listener.handler.tomb.Kill(nil)
					listener.handler.tomb.Wait()
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
					listener.handler.tomb.Go(func() error {
						return listener.handler.HandleConnection(conn, metricsChan)
					})
				}
			}
		}
	})
	listener.logger.Info("Moira Filter Listener Started")
	return metricsChan
}

// Stop stops listening connection
func (listener *MetricsListener) Stop() error {
	listener.tomb.Kill(nil)
	return listener.tomb.Wait()
}
