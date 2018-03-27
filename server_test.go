package wserver

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func Test_Server_1(t *testing.T) {
	port := 12345
	userID := uuid.New().String()
	event := "e1"

	count := 100

	s := NewServer(":" + strconv.Itoa(port))

	// run wserver
	go runBasicWServer(s)
	time.Sleep(time.Millisecond * 300)

	// push message
	registerCh := make(chan struct{})
	go func() {
		<-registerCh
		for i := 0; i < count; i++ {
			msg := fmt.Sprintf("hello -- %d", i)
			_, err := s.Push(userID, event, msg)
			if err != nil {
				t.Errorf("push message fail: %v", err)
			}

			// time.Sleep(time.Microsecond)
		}
	}()

	// dial websocket
	url := fmt.Sprintf("ws://127.0.0.1:%d/ws", port)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial websocket url fail: %v", err)
	}

	// register
	rm := RegisterMessage{
		Token: userID,
		Event: event,
	}
	if err := conn.WriteJSON(rm); err != nil {
		t.Fatalf("registe fail: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	close(registerCh)

	// read
	cnt := 0
	for {
		_, r, err := conn.NextReader()
		if err != nil {
			break
		}
		b, _ := ioutil.ReadAll(r)
		t.Logf("msg: %s", string(b))

		cnt++

		if cnt >= count {
			break
		}
	}
}

func runBasicWServer(s *Server) {
	if err := s.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
