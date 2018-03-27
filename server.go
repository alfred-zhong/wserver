package wserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	serverDefaultWSPath   = "/ws"
	serverDefaultSendPath = "/send"
)

var defaultUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(*http.Request) bool {
		return true
	},
}

// Server defines parameters for running websocket server.
type Server struct {
	// Path for websocket request, default "/ws".
	WSPath string

	// Path for send message, default "/send".
	SendPath string

	// Upgrader is for upgrade connection to websocket connection using
	// "github.com/gorilla/websocket".
	//
	// If Upgrader is nil, default upgrader will be used. Default upgrader is
	// set ReadBufferSize and WriteBufferSize to 1024, and CheckOrigin always
	// returns true.
	Upgrader *websocket.Upgrader

	// Check token if it's valid and return userID. If token is invalid, err
	// should not be nil.
	AuthToken func(token string) (userID string, err error)

	// Authorize send request. Message will be sent if it returns true,
	// otherwise the request will be discarded. Default nil and send request
	// will always be accepted.
	SendAuth func(r *http.Request) bool

	wh *websocketHandler
	sh *sendHandler
}

// Listen listens on the TCP network address addr.
func (s *Server) Listen(addr string) error {
	b := &binder{
		userID2EventConnMap: make(map[string]*[]eventConn),
		connID2UserIDMap:    make(map[string]string),
	}

	// websocket request handler
	wh := websocketHandler{
		upgrader: defaultUpgrader,
		binder:   b,
	}
	if s.Upgrader != nil {
		wh.upgrader = s.Upgrader
	}
	if s.AuthToken != nil {
		wh.calcUserIDFunc = s.AuthToken
	}
	s.wh = &wh
	http.Handle(s.WSPath, s.wh)

	// send request handler
	sh := sendHandler{
		binder: b,
	}
	if s.SendAuth != nil {
		sh.authFunc = s.SendAuth
	}
	s.sh = &sh
	http.Handle(s.SendPath, s.sh)

	return http.ListenAndServe(addr, nil)
}

// Push filters connections by userID and event, then write message
func (s *Server) Push(userID, event, message string) (int, error) {
	return s.sh.send(userID, event, message)
}

// Check parameters of Server, returns error if fail.
func (s Server) check() error {
	if !checkPath(s.WSPath) {
		return fmt.Errorf("WSPath: %s not illegal", s.WSPath)
	}
	if !checkPath(s.SendPath) {
		return fmt.Errorf("SendPath: %s not illegal", s.SendPath)
	}
	if s.WSPath == s.SendPath {
		return errors.New("WSPath is equal to SendPath")
	}

	return nil
}

// NewServer creates a new Server.
func NewServer() *Server {
	return &Server{
		WSPath:   serverDefaultWSPath,
		SendPath: serverDefaultSendPath,
	}
}

func checkPath(path string) bool {
	if path != "" && !strings.HasPrefix(path, "/") {
		return false
	}
	return true
}
