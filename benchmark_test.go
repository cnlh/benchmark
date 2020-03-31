package main

import (
	"io"
	"net/http"
	"net/http/httputil"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// create a http server
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		io.WriteString(writer, "work well!")
	})
	go http.ListenAndServe(":15342", nil)
	time.Sleep(time.Second)
	m.Run()
}

func TestBenchmark_Run(t *testing.T) {
	// create a request
	r, err := http.NewRequest("GET", "http://127.0.0.0.1:15342", nil)
	if err != nil {
		t.Fatal(err)
	}
	writeBytes, err := httputil.DumpRequest(r, true)
	if err != nil {
		t.Fatal(err)
	}
	p := &benchmark{
		connectionNum: 100,
		reqNum:        20000,
		requestBytes:  writeBytes,
		target:        "127.0.0.1:15342",
		schema:        r.URL.Scheme,
		timeout:       30000,
		reqConnList:   make([]*ReqConn, 0),
	}
	p.Run()
	p.Print()
}
