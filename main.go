package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strings"
)

var (
	header        = flag.String("h", "", "request header, split by \\r\\n")
	method        = flag.String("m", "GET", "request method")
	timeout       = flag.Int("t", 10000, "request/socket timeout in ms")
	connectionNum = flag.Int("c", 1000, "number of connection")
	requestNum    = flag.Int("n", 100000, "number of request")
	body          = flag.String("b", "", "ody of request")
	cpu           = flag.Int("cpu", 0, "number of cpu used")
	host          = flag.String("host", "", "host of request")
	proxyUrl      = flag.String("proxy", "", "proxy of request")
)

func main() {
	flag.Parse()
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
		for _, v := range strings.Split(*header, "\r\n") {
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

	p := &performance{
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
