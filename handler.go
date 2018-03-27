package wserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// websocketHandler defines to handle websocket upgrade request.
type websocketHandler struct {
	// upgrader is used to upgrade request.
	upgrader *websocket.Upgrader

	// binder stores relations about websocket connection and userID.
	binder *binder

	// calcUserIDFunc defines to calculate userID by token. The userID will
	// be equal to token if this function is nil.
	calcUserIDFunc func(token string) (userID string, err error)
}

// registerMessage defines message struct client send after connection
// to the server.
type registerMessage struct {
	Token string
	Event string
}

// First try to upgrade connection to websocket. If success, connection will
// be kept until client send close message.
func (wh *websocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wsConn, err := wh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer wsConn.Close()

	// handle Websocket request
	conn := NewConn(wsConn)
	conn.AfterReadFunc = func(messageType int, r io.Reader) {
		var rm registerMessage
		decoder := json.NewDecoder(r)
		if err := decoder.Decode(&rm); err != nil {
			return
		}

		// calculate userID by token
		userID := rm.Token
		if wh.calcUserIDFunc != nil {
			uID, err := wh.calcUserIDFunc(rm.Token)
			if err != nil {
				return
			}
			userID = uID
		}

		// bind
		wh.binder.Bind(userID, rm.Event, conn)
	}
	conn.BeforeCloseFunc = func() {
		// unbind
		wh.binder.Unbind(conn)
	}

	conn.Listen()
}

// ErrRequestIllegal describes error when data of the request is unaccepted.
var ErrRequestIllegal = errors.New("request data illegal")

// sendHandler defines to handle send message request.
type sendHandler struct {
	// authFunc defines to authorize request. The request will proceed only
	// when it returns true.
	authFunc func(r *http.Request) bool

	binder *binder
}

// Authorize if needed. Then decode the request and send message to each
// realted websocket connection.
func (s *sendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// authorize
	if s.authFunc != nil {
		if ok := s.authFunc(r); !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	// read request
	var sm sendMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&sm); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(ErrRequestIllegal.Error()))
		return
	}

	// validate the data
	if sm.UserID == "" || sm.Event == "" || sm.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(ErrRequestIllegal.Error()))
		return
	}

	cnt, err := s.send(sm.UserID, sm.Event, sm.Message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	result := strings.NewReader(fmt.Sprintf("message sent to %d clients", cnt))
	io.Copy(w, result)
}

func (s *sendHandler) send(userID, event, message string) (int, error) {
	if userID == "" || event == "" || message == "" {
		return 0, errors.New("parameters(userId, event, message) can't be empty")
	}

	// filter connections by userID and event, then send message
	conns, err := s.binder.FilterConn(userID, event)
	if err != nil {
		return 0, fmt.Errorf("filter conn fail: %v", err)
	}
	cnt := 0
	for i := range conns {
		_, err := conns[i].Write([]byte(message))
		if err != nil {
			s.binder.Unbind(conns[i])
			continue
		}
		cnt++
	}

	return cnt, nil
}

// sendMessage defines message struct send by client to push to each connected
// websocket client.
type sendMessage struct {
	UserID  string `json:"userId"`
	Event   string
	Message string
}
