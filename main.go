// Copyright 2020 The benchmark. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strings"
)

var (
	header        = flag.String("h", "", "request header, split by \\r\\n")
	method        = flag.String("m", "GET", "request method")
	timeout       = flag.Int("t", 10000, "request/socket timeout in ms")
	connectionNum = flag.Int("c", 1000, "number of connection")
	requestNum    = flag.Int("n", 100000, "number of request")
	body          = flag.String("b", "", "body of request")
	cpu           = flag.Int("cpu", 0, "number of cpu used")
	host          = flag.String("host", "", "host of request")
	proxyUrl      = flag.String("proxy", "", "proxy of request")
	ignoreErr     = flag.Bool("ignore-err", false, "`true` to ignore error when creating connection (default false)")
)

func main() {
	flag.Parse()
	if u, err := url.Parse(flag.Arg(0)); err != nil || u.Host == "" {
		fmt.Printf("the request url %s is not correct \n", flag.Arg(0))
		return
	}
	payload := strings.NewReader(*body)
	req, err := http.NewRequest(*method, flag.Arg(0), payload)
	if err != nil {
		fmt.Println(err)
		return
	}
	var target = req.Host
	if *host != "" {
		req.Host = *host
	}
	if *cpu > 0 {
		runtime.GOMAXPROCS(*cpu)
	}
	if *header != "" {
		for _, v := range strings.Split(*header, "\\r\\n") {
			a := strings.Split(v, ":")
			if len(a) == 2 {
				req.Header.Set(strings.Trim(a[0], " "), strings.Trim(a[1], " "))
			}
		}
	}
	writeBytes, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !strings.Contains(target, ":") {
		if req.URL.Scheme == "http" {
			target = target + ":80"
		} else {
			target = target + ":443"
		}
	}

	p := &benchmark{
		connectionNum: *connectionNum,
		reqNum:        int64(*requestNum),
		requestBytes:  writeBytes,
		target:        target,
		schema:        req.URL.Scheme,
		timeout:       *timeout,
		reqConnList:   make([]*ReqConn, 0),
		proxy:         *proxyUrl,
	}
	p.Run()
	p.Print()
}
