package connection

import (
	"fmt"
	"net"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/filter"
	"github.com/moira-alert/moira/metrics/graphite"
)

// MetricsListener is facade for standard net.MetricsListener and accept connection for handling it
type MetricsListener struct {
	listener *net.TCPListener
	handler  *Handler
	logger   moira.Logger
	tomb     tomb.Tomb
	metrics  *graphite.FilterMetrics
}

// NewListener creates new listener
func NewListener(port string, logger moira.Logger, metrics *graphite.FilterMetrics, patternStorage *filter.PatternStorage) (*MetricsListener, error) {
	address, err := net.ResolveTCPAddr("tcp", port)
	if nil != err {
		return nil, fmt.Errorf("Failed to resolve tcp address [%s]: %s", port, err.Error())
	}
	newListener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("Failed to listen on [%s]: %s", port, err.Error())
	}
	listener := MetricsListener{
		listener: newListener,
		logger:   logger,
		handler:  NewConnectionsHandler(logger, patternStorage),
		metrics: 	metrics,
	}
	return &listener, nil
}

// Listen waits for new data in connection and handles it in ConnectionHandler
// All handled data sets to metricsChan
func (listener *MetricsListener) Listen() chan *moira.MatchedMetric {
	metricsChan := make(chan *moira.MatchedMetric, 16384)
	listener.tomb.Go(func() error {
		for {
			select {
			case <-listener.tomb.Dying():
				{
					listener.logger.Info("Stopping listener...")
					listener.listener.Close()
					listener.handler.StopHandlingConnections()
					close(metricsChan)
					listener.logger.Info("Moira Filter Listener stopped")
					return nil
				}
			default:
			}
			listener.listener.SetDeadline(time.Now().Add(1e9))
			conn, err := listener.listener.Accept()
			if nil != err {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				listener.logger.Infof("Failed to accept connection: %s", err.Error())
				continue
			}
			listener.logger.Infof("%s connected", conn.RemoteAddr())
			listener.handler.HandleConnection(conn, metricsChan)
		}
	})

	listener.tomb.Go(func() error { return listener.checkNewMetricsChannelLen(metricsChan) })
	listener.logger.Info("Moira Filter Listener Started")
	return metricsChan
}

func (listener *MetricsListener) checkNewMetricsChannelLen(channel <-chan *moira.MatchedMetric) error {
	checkTicker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-listener.tomb.Dying():
			return nil
		case <-checkTicker.C:
			listener.metrics.MetricChannelLen.Update(int64(len(channel)))
		}
	}
}

// Stop stops listening connection
func (listener *MetricsListener) Stop() error {
	listener.tomb.Kill(nil)
	return listener.tomb.Wait()
}
