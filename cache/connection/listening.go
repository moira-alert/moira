package connection

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/cache"
	"net"
	"sync"
)

//MetricsListener is facade for standard net.MetricsListener and accept connection for handling it
type MetricsListener struct {
	listener net.Listener
	handler  *Handler
	logger   moira.Logger
}

//NewListener creates new listener
func NewListener(port string, logger moira.Logger, patternStorage *cache.PatternStorage) (*MetricsListener, error) {
	listen := port
	newListener, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, fmt.Errorf("Failed to listen on [%s]: %s", listen, err.Error())
	}
	listener := MetricsListener{
		listener: newListener,
		handler:  NewConnectionHandler(logger, patternStorage),
	}
	return &listener, nil
}

//Listen waits for new data in connection and handles it in ConnectionHandler
//All handled data sets to metricsChan
func (listener *MetricsListener) Listen(metricsChan chan *moira.MatchedMetric, wg *sync.WaitGroup, shutdown chan bool) {
	defer wg.Done()
	var handlerWG sync.WaitGroup
	for {
		select {
		case <-shutdown:
			{
				listener.logger.Info("Stop listen connection")
				handlerWG.Wait()
				close(metricsChan)
				break
			}
		default:
			{
				conn, err := listener.listener.Accept()
				if err != nil {
					listener.logger.Infof("Failed to accept connection: %s", err.Error())
					continue
				}
				handlerWG.Add(1)
				go listener.handler.HandleConnection(conn, metricsChan, shutdown, &handlerWG)
			}
		}
	}
}
