package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	header        = flag.String("h", "", "request header, split by \\r\\n")
	method        = flag.String("m", "GET", "request method")
	timeout       = flag.Int("t", 3000, "request/socket timeout in ms")
	connectionNum = flag.Int("c", 1000, "the number of connection")
	requestNum    = flag.Int("n", 100000, "the number of request")
	body          = flag.String("b", "", "the body of request")
	cpu           = flag.Int("cpu", 0, "the number of cpu used")
	host          = flag.String("host", "", "the host of request")
	//默认失败次数
)

var (
	rawBytes = []byte("\r\n\r\n")
	rowBytes = []byte("\r\n")
	lenBytes = []byte("Content-Length: ")
	l        = len(lenBytes)
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
	var wg sync.WaitGroup
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

	var requestNumInt64 = int64(*requestNum)
	var nowNum int64
	var reqConnList []*ReqConn
	wg.Add(*connectionNum)
	timeStart := time.Now()
	for i := 0; i < *connectionNum; i++ {
		rc := &ReqConn{
			Count:      requestNumInt64,
			NowNum:     &nowNum,
			timeout:    *timeout,
			writeBytes: writeBytes,
			reqTimes:   make([]time.Time, 0),
			remoteAddr: target,
			schema:     req.URL.Scheme,
			buf:        make([]byte, 65535),
		}
		go func() {
			if err = rc.Start(); err != nil {
				fmt.Println(err.Error())
				os.Exit(0)
			}
			wg.Done()
		}()
		reqConnList = append(reqConnList, rc)
	}
	wg.Wait()
	fmt.Println(float64(*requestNum) / (time.Now().Sub(timeStart).Seconds()))
}

func Bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

type ReqConn struct {
	ErrorTimes int
	Count      int64
	NowNum     *int64
	timeout    int
	writeBytes []byte
	writeLen   int
	readLen    int
	reqTimes   []time.Time
	conn       net.Conn
	remoteAddr string
	schema     string
	buf        []byte
}

func (rc *ReqConn) dial() error {
	var err error
	rc.conn, err = net.DialTimeout("tcp", rc.remoteAddr, time.Millisecond*time.Duration(rc.timeout))
	if err != nil {
		return err
	}
	if rc.schema == "https" {
		conf := &tls.Config{
			InsecureSkipVerify: true,
		}
		rc.conn = tls.Client(rc.conn, conf)
	}
	return nil
}

func (rc *ReqConn) Start() (err error) {
	var contentLen string
	var hasRead int
	var n int
re:
	if err != io.EOF {
		rc.ErrorTimes += 1
	}
	if err = rc.dial(); err != nil {
		return
	}
	for {
		hasRead = 0
		n, err = rc.conn.Write(rc.writeBytes)
		if err != nil {
			goto re
		}
		rc.writeLen += n
		n, err = rc.conn.Read(rc.buf)
		if err != nil {
			goto re
		}
		rc.readLen += n
		bbArr := bytes.SplitN(rc.buf[:n], rawBytes, 2)
		n := bytes.Index(bbArr[0], lenBytes)
		start := n + l
		end := bytes.Index(bbArr[0][start:], rowBytes)
		if end == -1 {
			contentLen = Bytes2str(bbArr[0][start:])
		} else {
			contentLen = Bytes2str(bbArr[0][start : start+end])
		}
		contentLenI, _ := strconv.Atoi(contentLen)
		hasRead += len(bbArr[1])
		if hasRead < contentLenI {
			for {
				if hasRead >= contentLenI {
					break
				}
				n, err = rc.conn.Read(rc.buf)
				if err != nil {
					goto re
				}
				rc.readLen += n
				hasRead += n
			}
		}
		if atomic.AddInt64(rc.NowNum, 1) >= rc.Count {
			return
		}
	}
}
