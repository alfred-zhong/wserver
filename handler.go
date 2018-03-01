package wserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// websocketHandler defines to handle websocket upgrade request.
type websocketHandler struct {
	upgrader *websocket.Upgrader
}

// ServeHTTP will first try to upgrade connection to websocket. If success,
// connection will be kept until client send close message.
func (wh *websocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := wh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// handle Websocket request
	dataCh := make(chan []byte)
	wsConn := NewConn(conn, dataCh)
	// TODO: set AfterReadFunc and BeforeCloseFunc

	wsConn.Listen()
}

type sendHandler struct {
	SendHeadKey   string
	SendHeadValue string
}

func (s *sendHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 验证头信息
	if r.Header.Get(s.SendHeadKey) != s.SendHeadValue {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("头信息 %s 值不正确", s.SendHeadKey)))
		return
	}

	// 读取请求
	var sMsg sendMsg
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&sMsg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("数据不合法"))
		return
	}

	// 验证请求数据
	if sMsg.EmployeeID <= 0 || sMsg.Event == "" || sMsg.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("数据不合法"))
		return
	}

	// 发送请求
	go sendMessage(sMsg.EmployeeID, sMsg.Event, []byte(sMsg.Message))

	w.Write([]byte("推送成功"))
}

type sendMsg struct {
	EmployeeID int64 `json:"employeeId"`
	Event      string
	Message    string
}
