package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func NewSubscribers() *Subscribers {
	ss := &Subscribers{
		subscribers:      make(map[*Subscriber]bool),
		requests:         make(map[string]chan<- Response),
		mtx:              sync.Mutex{},
		InCh:             make(chan Response, 10),
		addSubscriber:    make(chan *Subscriber),
		removeSubscriber: make(chan *Subscriber),
		sendRequest:      make(chan subscriberRequest, 10),
		removeRequest:    make(chan Request, 10),
	}
	go ss.shadowWorker()
	return ss
}

type Subscribers struct {
	subscribers      map[*Subscriber]bool
	requests         map[string]chan<- Response
	mtx              sync.Mutex
	InCh             chan Response
	addSubscriber    chan *Subscriber
	removeSubscriber chan *Subscriber
	sendRequest      chan subscriberRequest
	removeRequest    chan Request
}

func (ss *Subscribers) Add(s *Subscriber) {
	ss.addSubscriber <- s
}

func (ss *Subscribers) Remove(s *Subscriber) {
	ss.removeSubscriber <- s
}

func (ss *Subscribers) Handler(ctx context.Context, req Request) (Response, error) {
	ch := make(chan Response)
	req.RequestID = genRequestID()
	ss.sendRequest <- subscriberRequest{outCh: ch, req: req}
	select {
	case resp := <-ch:
		close(ch)
		return resp, nil
	case <-ctx.Done():
		ss.removeRequest <- req
		close(ch)
		return Response{}, ctx.Err()
	}
}

func (ss *Subscribers) shadowWorker() {
	for {
		select {
		case resp := <-ss.InCh:
			ch, ok := ss.requests[resp.RequestID]
			if ok {
				delete(ss.requests, resp.RequestID)
				ch <- resp
			}
		case s := <-ss.addSubscriber:
			ss.subscribers[s] = true
		case s := <-ss.removeSubscriber:
			delete(ss.subscribers, s)
		case sreq := <-ss.sendRequest:
			ss.requests[sreq.req.RequestID] = sreq.outCh

			for s := range ss.subscribers {
				s.Write(sreq.req)
			}
		case req := <-ss.removeRequest:
			delete(ss.requests, req.RequestID)
		}
	}
}

type subscriberRequest struct {
	outCh chan<- Response
	req   Request
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 10240
)

func NewSubscriber(conn *websocket.Conn, ss *Subscribers) *Subscriber {

	s := &Subscriber{
		conn: conn,
		ss:   ss,
		send: make(chan Request, 1),
		done: make(chan struct{}),
		once: sync.Once{},
	}
	conn.SetCloseHandler(func(code int, text string) error {
		s.close()
		return nil
	})
	go s.writeLoop()
	go s.readLoop()
	return s
}

type Subscriber struct {
	conn *websocket.Conn
	ss   *Subscribers
	send chan Request
	done chan struct{}
	once sync.Once
}

func (s *Subscriber) close() {
	s.once.Do(func() {
		s.ss.Remove(s)
		close(s.done)
		s.conn.Close()
	})
}

func (s *Subscriber) Write(req Request) error {
	s.send <- req
	return nil
}

func (s *Subscriber) readLoop() {
	defer s.close()

	for {
		var req Response
		err := s.conn.ReadJSON(&req)
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				log.Println(err)
			}
			break
		}
		s.ss.InCh <- req
	}
}

func (s *Subscriber) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		s.close()
	}()
	for {
		select {
		case request, ok := <-s.send:
			if !ok {
				return
			}
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := s.conn.WriteJSON(request)
			if err != nil {
				log.Println(err)
				return
			}
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println(err)
				// skip error because client can close connection without notify us
				return
			}
		case <-s.done:
			return
		}
	}
}
