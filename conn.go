package wserver

import (
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Conn wraps websocket.Conn with Conn. It defines to listen and read
// data from Conn.
type Conn struct {
	Conn *websocket.Conn

	AfterReadFunc   func(messageType int, r io.Reader)
	BeforeCloseFunc func()

	once   sync.Once
	id     string
	stopCh chan struct{}
}

// Write write p to the websocket connection. The error returned will always
// be nil if success.
func (c *Conn) Write(p []byte) (n int, err error) {
	err = c.Conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// GetID returns the id generated using UUID algorithm.
func (c *Conn) GetID() string {
	c.once.Do(func() {
		u := uuid.New()
		c.id = u.String()
	})

	return c.id
}

// Listen listens for receive data from websocket connection. It blocks
// until websocket connection is closed.
func (c *Conn) Listen() (err error) {
	c.Conn.SetCloseHandler(func(code int, text string) error {
		if c.BeforeCloseFunc != nil {
			c.BeforeCloseFunc()
		}

		message := websocket.FormatCloseMessage(code, "")
		c.Conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		return nil
	})

	c.read()
	return nil
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
func NewConn(conn *websocket.Conn) *Conn {
	return &Conn{
		Conn:   conn,
		stopCh: make(chan struct{}),
	}
}
