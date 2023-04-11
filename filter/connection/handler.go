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
	terminate chan struct{}
}

// NewConnectionsHandler creates new Handler
func NewConnectionsHandler(logger moira.Logger) *Handler {
	return &Handler{
		logger:    logger,
		terminate: make(chan struct{}, 1),
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
	closeConnection := make(chan struct{})
	go func(conn net.Conn) {
		select {
		case <-handler.terminate:
			conn.Close()
		case <-closeConnection:
		}
	}(connection)

	for {
		bytes, err := buffer.ReadBytes('\n')
		if err != nil {
			connection.Close()
			if err != io.EOF {
				handler.logger.Error().
					Error(err).
					Msg("Fail to read from metric connection")
			}
			close(closeConnection)
			return
		}
		bytesWithoutCRLF := dropCRLF(bytes)
		if len(bytesWithoutCRLF) > 0 {
			lineChan <- bytesWithoutCRLF
		}
	}
}

// StopHandlingConnections closes all open connections and wait for handling remaining metrics
func (handler *Handler) StopHandlingConnections() {
	close(handler.terminate)
	handler.wg.Wait()
}

func dropCRLF(bytes []byte) []byte {
	bytesLength := len(bytes)
	if bytesLength > 0 && bytes[bytesLength-1] == '\n' {
		bytesLength--
	}
	if bytesLength > 0 && bytes[bytesLength-1] == '\r' {
		bytesLength--
	}
	return bytes[:bytesLength]
}
