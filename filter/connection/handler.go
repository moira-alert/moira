package connection

import (
	"bufio"
	"io"
	"net"
	"sync"

	"github.com/moira-alert/moira"
)

// Handler handling connection data and shift it to lineChan channel
type Handler struct {
	logger    moira.Logger
	wg        sync.WaitGroup
	terminate chan bool
}

// NewConnectionsHandler creates new Handler
func NewConnectionsHandler(logger moira.Logger) *Handler {
	return &Handler{
		logger:    logger,
		terminate: make(chan bool, 1),
	}
}

// HandleConnection convert every line from connection to metric and send it to lineChan channel
func (handler *Handler) HandleConnection(connection net.Conn, lineChan chan<- []byte) {
	handler.wg.Add(1)
	go func() {
		defer handler.wg.Done()
		handler.handle(connection, lineChan)
	}()
}

func (handler *Handler) handle(connection net.Conn, lineChan chan<- []byte) {
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
		lineChan <- lineBytes
	}
}

// StopHandlingConnections closes all open connections and wait for handling ramaining metrics
func (handler *Handler) StopHandlingConnections() {
	close(handler.terminate)
	handler.wg.Wait()
}
