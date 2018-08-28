package stub

import "strings"

type methodMatcher string

func (m methodMatcher) Match(r *Request) bool {
	return strings.ToUpper(string(m)) == strings.ToUpper(r.Method)
}

type headerMatcher struct {
	k, v string
}

func (m headerMatcher) Match(r *Request) bool {
	v := r.Header.Get(m.k)
	return v == m.v
}

type pathMatcher string

func (m pathMatcher) Match(r *Request) bool {
	return strings.HasPrefix(r.URL.Path, string(m))
}

type bodyMatcher func(body []byte) bool

func (m bodyMatcher) Match(r *Request) bool {
	return m(r.Body)
}
