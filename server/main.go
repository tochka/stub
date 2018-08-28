package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/namsral/flag"
)

var (
	waitTimeout time.Duration
	port        int
)

func init() {
	flag.DurationVar(&waitTimeout, "wait-timeout", time.Second, "Waiting response timeout")
	flag.IntVar(&port, "port", 8080, "Address for server")
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 10 * time.Second,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)

	ss := NewSubscribers()

	http.HandleFunc("/", Consumer(ss.Handler))
	http.HandleFunc("/subscribez", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ERROR] Cannot upgrade connection: %+v\n", err)
			return
		}
		s := NewSubscriber(conn, ss)

		ss.Add(s)
	})

	log.Println("Server started ...")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}

type Request struct {
	Method    string      `json:"method"`
	URL       string      `json:"url"`
	Body      []byte      `json:"body"`
	Header    http.Header `json:"header"`
	RequestID string      `json:"request_id"`
}

type Response struct {
	Status    int         `json:"status"`
	Header    http.Header `json:"header"`
	Body      []byte      `json:"body"`
	RequestID string      `json:"request_id"`
}
