package stub

import (
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func NewConnect(address string) (*Connect, error) {
	conn, _, err := websocket.DefaultDialer.Dial(address+"/subscribez", nil)
	if err != nil {
		return nil, err
	}

	c := &Connect{
		conn:       conn,
		addStub:    make(chan *Stub),
		removeStub: make(chan *Stub),
		newReq:     make(chan *internalRequest, 10),
		done:       make(chan struct{}),
	}

	go c.readLoop()
	go c.loop()

	return c, nil
}

type Connect struct {
	conn       *websocket.Conn
	addStub    chan *Stub
	removeStub chan *Stub
	newReq     chan *internalRequest
	done       chan struct{}
}

func (c *Connect) Stub() *Stub {
	return &Stub{con: c}
}

func (c *Connect) Close() {
	c.done <- struct{}{}
	c.conn.WriteControl(websocket.CloseMessage, nil, time.Now().Add(10*time.Second))
	c.conn.Close()
}

func (c *Connect) loop() {
	stubs := make(map[*Stub]bool)
	for {
		select {
		case s := <-c.addStub:
			stubs[s] = true
		case s := <-c.removeStub:
			delete(stubs, s)
		case r := <-c.newReq:
			for s := range stubs {
				u, err := url.Parse("http://localhost" + r.URL)
				if err != nil {
					log.Println("Cannot parse url", r.URL)
					break
				}
				req := &Request{
					Body:   r.Body,
					Method: r.Method,
					Header: r.Header,
					URL:    u,
				}
				if s.match(req) {
					iresp := internalResponse{
						Status:    s.result.Status,
						Body:      s.result.Body,
						Header:    s.result.Header,
						RequestID: r.RequestID,
					}
					err := c.conn.WriteJSON(iresp)
					if err != nil {
						log.Println("Write error req:", r.RequestID, " err:", err)
					}
				}
			}
		case <-c.done:
			return
		}
	}
}

func (c *Connect) readLoop() {
	for {
		req := new(internalRequest)
		err := c.conn.ReadJSON(req)
		if err != nil {
			log.Println("Error:", err)
			break
		}
		c.newReq <- req
	}
}

type internalRequest struct {
	Body      []byte      `json:"body"`
	Method    string      `json:"method"`
	Header    http.Header `json:"header"`
	URL       string      `json:"url"`
	RequestID string      `json:"request_id"`
}

type internalResponse struct {
	Status    int         `json:"status"`
	Body      []byte      `json:"body"`
	Header    http.Header `json:"header"`
	RequestID string      `json:"request_id"`
}
