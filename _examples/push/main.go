package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alfred-zhong/wserver"
)

func main() {
	pushURL := "http://127.0.0.1:12345/push"
	contentType := "application/json"

	for {
		pm := wserver.PushMessage{
			UserID:  "jack",
			Event:   "topic1",
			Message: fmt.Sprintf("Hello in %s", time.Now().Format("2006-01-02 15:04:05.000")),
		}
		b, _ := json.Marshal(pm)

		http.DefaultClient.Post(pushURL, contentType, bytes.NewReader(b))

		time.Sleep(time.Second)
	}
}
