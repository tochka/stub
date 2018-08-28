package stub

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type Stub struct {
	con      *Connect
	matchers []Matcher
	result   Response
}

func (s *Stub) match(r *Request) bool {
	for _, m := range s.matchers {
		if !m.Match(r) {
			return false
		}
	}
	return true
}

type Response struct {
	Status int
	Body   []byte
	Header http.Header
}

type Matcher interface {
	Match(r *Request) bool
}

type Request struct {
	Body   []byte
	Method string
	Header http.Header
	URL    *url.URL
}

func (s *Stub) Return(resp Response) ReleaseStub {
	s.result = resp
	s.con.addStub <- s
	return ReleaseStub{s}
}

func (s *Stub) ReturnJSON(code int, payload interface{}) ReleaseStub {
	b, _ := json.Marshal(payload)
	s.result = Response{
		Status: code,
		Body:   b,
	}
	s.con.addStub <- s
	return ReleaseStub{s}
}

func (s *Stub) MethodPost() *Stub {
	s.matchers = append(s.matchers, methodMatcher(http.MethodPost))
	return s
}

func (s *Stub) MethodGet() *Stub {
	s.matchers = append(s.matchers, methodMatcher(http.MethodGet))
	return s
}
func (s *Stub) MethodPatch() *Stub {
	s.matchers = append(s.matchers, methodMatcher(http.MethodPatch))
	return s
}

func (s *Stub) MethodDelete() *Stub {
	s.matchers = append(s.matchers, methodMatcher(http.MethodDelete))
	return s
}

func (s *Stub) MethodPut() *Stub {
	s.matchers = append(s.matchers, methodMatcher(http.MethodPut))
	return s
}

func (s *Stub) Header(k, v string) *Stub {
	s.matchers = append(s.matchers, headerMatcher{k, v})
	return s
}

func (s *Stub) Path(p string) *Stub {
	s.matchers = append(s.matchers, pathMatcher(p))
	return s
}

func (s *Stub) Body(m func(body []byte) bool) *Stub {
	s.matchers = append(s.matchers, bodyMatcher(m))
	return s
}

type ReleaseStub struct {
	s *Stub
}

func (rs ReleaseStub) Remove() {
	rs.s.con.removeStub <- rs.s
}
