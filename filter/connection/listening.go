package connection

import (
	"fmt"
	"net"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/metrics"
)

// MetricsListener is facade for standard net.MetricsListener and accept connection for handling it
type MetricsListener struct {
	listener *net.TCPListener
	handler  *Handler
	logger   moira.Logger
	tomb     tomb.Tomb
	metrics  *metrics.FilterMetrics
}

// NewListener creates new listener
func NewListener(port string, logger moira.Logger, metrics *metrics.FilterMetrics) (*MetricsListener, error) {
	address, err := net.ResolveTCPAddr("tcp", port)
	if nil != err {
		return nil, fmt.Errorf("failed to resolve tcp address [%s]: %s", port, err.Error())
	}
	newListener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on [%s]: %s", port, err.Error())
	}
	listener := MetricsListener{
		listener: newListener,
		logger:   logger,
		handler:  NewConnectionsHandler(logger),
		metrics:  metrics,
	}
	return &listener, nil
}

// Listen waits for new data in connection and handles it in ConnectionHandler
// All handled data sets to lineChan
func (listener *MetricsListener) Listen() chan []byte {
	lineChan := make(chan []byte, 16384) //nolint
	listener.tomb.Go(func() error {
		for {
			select {
			case <-listener.tomb.Dying():
				{
					listener.logger.Info().Msg("Stopping listener...")
					listener.listener.Close()
					listener.handler.StopHandlingConnections()
					close(lineChan)
					listener.logger.Info().Msg("Moira Filter Listener stopped")
					return nil
				}
			default:
			}
			listener.listener.SetDeadline(time.Now().Add(1e9)) //nolint
			conn, err := listener.listener.Accept()
			if nil != err {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				listener.logger.Info().
					Error(err).
					Msg("Failed to accept connection")
				continue
			}
			listener.logger.Info().
				String("remote_address", conn.RemoteAddr().String()).
				Msg("Someone connected")

			listener.handler.HandleConnection(conn, lineChan)
		}
	})
	listener.tomb.Go(func() error { return listener.checkNewLinesChannelLen(lineChan) })
	listener.logger.Info().Msg("Moira Filter Listener Started")
	return lineChan
}

func (listener *MetricsListener) checkNewLinesChannelLen(channel <-chan []byte) error {
	checkTicker := time.NewTicker(time.Millisecond * 100) //nolint
	for {
		select {
		case <-listener.tomb.Dying():
			return nil
		case <-checkTicker.C:
			listener.metrics.LineChannelLen.Update(int64(len(channel)))
		}
	}
}

// Stop stops listening connection
func (listener *MetricsListener) Stop() error {
	listener.tomb.Kill(nil)
	return listener.tomb.Wait()
}
