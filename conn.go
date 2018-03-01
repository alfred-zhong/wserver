package wserver

import (
	"io"
	"time"

	"github.com/gorilla/websocket"
)

// Conn wraps websocket.Conn with Conn. It defines to receive data from
// DataChan and write to Conn. Also it will listen and read data from Conn.
type Conn struct {
	Conn     *websocket.Conn
	DataChan chan []byte

	AfterReadFunc   func(messageType int, r io.Reader)
	BeforeCloseFunc func()

	stopCh chan struct{}
}

// Listen keeps receiving data from DataChan and writes it to the websocket
// connection. It returns until the DataChan closed or get an error from
// reading from websocket connection. The error returned is not nil when write
// data to websocket connection fails.
func (c *Conn) Listen() (err error) {
	c.Conn.SetCloseHandler(func(code int, text string) error {
		if c.BeforeCloseFunc != nil {
			c.BeforeCloseFunc()
		}

		message := websocket.FormatCloseMessage(code, "")
		c.Conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		return nil
	})

	go c.read()

ReceiveFromChan:
	for {
		select {
		case <-c.stopCh:
			break ReceiveFromChan
		case data, ok := <-c.DataChan:
			if ok {
				// TODO: messageType may be customized
				err = c.Conn.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					break ReceiveFromChan
				}
			} else {
				break ReceiveFromChan
			}
		}
	}

	return
}

// Keeps reading from Conn util get error.
func (c *Conn) read() {
	for {
		messageType, r, err := c.Conn.NextReader()
		if err != nil {
			// TODO: handle read error maybe
			break
		}

		if c.AfterReadFunc != nil {
			c.AfterReadFunc(messageType, r)
		}
	}

	close(c.stopCh)
}

// NewConn wraps conn.
func NewConn(conn *websocket.Conn, dataChan chan []byte) *Conn {
	return &Conn{
		Conn:     conn,
		DataChan: dataChan,
		stopCh:   make(chan struct{}),
	}
}
