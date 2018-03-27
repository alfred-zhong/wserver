package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alfred-zhong/wserver"
	"github.com/google/uuid"
)

func main() {
	port := 12345
	userID := uuid.New().String()
	event := "alarm"

	// run wserver
	go func() {
		s := wserver.NewServer(":" + strconv.Itoa(port))
		if err := s.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	// run http server
	go func() {
		httpPort := 8081

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			tplName := "index.html"
			t, err := template.ParseFiles(tplName)
			if err != nil {
				log.Printf("template[%s] parse fail: %v", tplName, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			t.Execute(w, struct {
				Port  int
				Token string
				Event string
			}{
				port,
				userID,
				event,
			})
		})

		log.Printf("open url: http://localhost:%d", httpPort)
		if err := http.ListenAndServe(":"+strconv.Itoa(httpPort), nil); err != nil {
			log.Fatalln(err)
		}
	}()

	time.Sleep(time.Second)

	// send messages in loop
	url := fmt.Sprintf("http://localhost:%d/send", port)
	for {
		msg := msg{
			UserID:  userID,
			Event:   event,
			Message: fmt.Sprintf("hello -- %d", rand.Int()),
		}
		sendMsg(msg, url)

		time.Sleep(10 * time.Millisecond)
	}
}

type msg struct {
	UserID  string `json:"userId"`
	Event   string `json:"event"`
	Message string `json:"message"`
}

func (m msg) String() string {
	bb, _ := json.Marshal(m)
	return string(bb)
}

func sendMsg(m msg, url string) {
	res, err := http.DefaultClient.Post(url, "application/json", strings.NewReader(m.String()))
	if err != nil {
		log.Printf("send message fail: %v", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("status code: %d not expected", res.StatusCode)
		return
	}
}
