package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"
)

func Consumer(hanlder func(ctx context.Context, req Request) (Response, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), waitTimeout)
		defer cancel()

		resp, err := hanlder(ctx, Request{
			Method: r.Method,
			URL:    r.RequestURI,
			Body:   b,
			Header: r.Header,
		})
		if err != nil {
			if err.Error() == context.DeadlineExceeded.Error() {
				w.WriteHeader(http.StatusRequestTimeout)
				return
			}
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for k, vals := range resp.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.Status)
		w.Write(resp.Body)
	}
}

func genRequestID() string {
	uid := make([]byte, 16)
	rand.Read(uid)
	return hex.EncodeToString(uid)
}
