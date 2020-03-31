package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"golang.org/x/net/proxy"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	rawBytes = []byte("\r\n\r\n")
	rowBytes = []byte("\r\n")
	lenBytes = []byte("Content-Length: ")
	l        = len(lenBytes)
)

type ReqConn struct {
	ErrorTimes int
	Count      int64
	NowNum     *int64
	timeout    int
	writeBytes []byte
	writeLen   int
	readLen    int
	reqTimes   []int
	conn       net.Conn
	remoteAddr string
	schema     string
	buf        []byte
	proxy      string
}

func (rc *ReqConn) dial() error {
	if rc.conn != nil {
		rc.conn.Close()
	}
	var err error
	var conn net.Conn
	if rc.proxy != "" {
		var u *url.URL
		var d proxy.Dialer
		u, err = url.Parse(rc.proxy)
		if err != nil {
			return err
		}
		switch u.Scheme {
		case "socks5":
			d, err = proxy.FromURL(u, nil)
			if err != nil {
				return err
			}
			conn, err = d.Dial("tcp", rc.remoteAddr)
		default:
			conn, err = NewHttpProxyConn(u, rc.remoteAddr)
		}

	} else {
		conn, err = net.DialTimeout("tcp", rc.remoteAddr, time.Millisecond*time.Duration(rc.timeout))
	}
	if err != nil {
		return err
	}
	rc.conn = conn
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
	var bodyHasRead int
	var headerHasRead int
	var n int
	var reqTime time.Time
re:
	if err != nil && err != io.EOF {
		rc.ErrorTimes += 1
	}
	if err = rc.dial(); err != nil {
		return
	}
	for {
		bodyHasRead = 0
		headerHasRead = 0
		reqTime = time.Now()
		n, err = rc.conn.Write(rc.writeBytes)
		if err != nil {
			goto re
		}
		rc.writeLen += n
	readHeader:
		rc.conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(rc.timeout)))
		n, err = rc.conn.Read(rc.buf[headerHasRead:])
		if err != nil {
			goto re
		}
		headerHasRead += n
		rc.readLen += n
		var bbArr [2][]byte
		bodyPos := bytes.Index(rc.buf[:headerHasRead], rawBytes)
		if bodyPos > -1 {
			bbArr[0] = rc.buf[:bodyPos]
			bbArr[1] = rc.buf[bodyPos+len(rawBytes):]
		} else {
			goto readHeader
		}
		n := bytes.Index(bbArr[0], lenBytes)
		start := n + l
		end := bytes.Index(bbArr[0][start:], rowBytes)
		if end == -1 {
			contentLen = Bytes2str(bbArr[0][start:])
		} else {
			contentLen = Bytes2str(bbArr[0][start : start+end])
		}
		contentLenI, _ := strconv.Atoi(contentLen)
		bodyHasRead += len(bbArr[1])
		for {
			if bodyHasRead >= contentLenI {
				break
			}
			n, err = rc.conn.Read(rc.buf)
			if err != nil {
				goto re
			}
			rc.readLen += n
			bodyHasRead += n
		}
		rc.reqTimes = append(rc.reqTimes, int(time.Now().Sub(reqTime).Milliseconds()))
		if atomic.AddInt64(rc.NowNum, 1) >= rc.Count {
			return
		}
	}
}

func Bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func NewHttpProxyConn(url *url.URL, remoteAddr string) (net.Conn, error) {
	req, err := http.NewRequest("CONNECT", "http://"+remoteAddr, nil)
	if err != nil {
		return nil, err
	}
	password, _ := url.User.Password()
	req.SetBasicAuth(url.User.Username(), password)
	// we make a http proxy request
	proxyConn, err := net.Dial("tcp", url.Host)
	if err != nil {
		return nil, err
	}
	if err := req.Write(proxyConn); err != nil {
		return nil, err
	}
	res, err := http.ReadResponse(bufio.NewReader(proxyConn), req)
	if err != nil {
		return nil, err
	}
	_ = res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New("Proxy error " + res.Status)
	}
	return proxyConn, nil
}
