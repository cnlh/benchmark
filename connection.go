// Copyright 2020 The benchmark. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// ReqConn is used to create a connection and record data
type ReqConn struct {
	ErrorTimes int
	Count      int64
	NowNum     *int64
	FailedNum  atomic.Int64
	timeout    int
	writeLen   int
	readLen    int
	reqTimes   []int
	conn       net.Conn
	readWriter ConnReadWriter
	remoteAddr string
	schema     string
	dialer     ProxyConn
}

// Connect to the server, http and socks5 proxy support
// If the target is https, convert connection to tls client
func (rc *ReqConn) dial() error {
	if rc.conn != nil {
		rc.conn.Close()
	}
	conn, err := rc.dialer.Dial("tcp", rc.remoteAddr, time.Millisecond*time.Duration(rc.timeout))
	if err != nil {
		return err
	}
	rc.conn = conn
	if rc.schema == "https" {
		var h string
		h, _, err = net.SplitHostPort(rc.remoteAddr)
		if err != nil {
			return err
		}
		conf := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         h,
		}
		rc.conn = tls.Client(rc.conn, conf)
	}
	return nil
}

// Start a connection, send request to server and read response from server
func (rc *ReqConn) Start() (err error) {
	var n int
	var reqTime time.Time
re:
	if err != nil && err != io.EOF && !strings.Contains(err.Error(), "connection reset by peer") {
		rc.ErrorTimes += 1
	}
	if rc.FailedNum.Load() >= rc.Count {
		fmt.Println("Test aborted due to too many errors, last error:", err)
		return
	}
	if err = rc.dial(); err != nil {
		return
	}
	for {
		reqTime = time.Now()
		rc.conn.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(rc.timeout)))
		n, err = rc.readWriter.Write(rc.conn)
		if err != nil {
			rc.FailedNum.Add(1)
			goto re
		}
		rc.writeLen += n
		n, err = rc.readWriter.Read(rc.conn)
		if err != nil {
			rc.FailedNum.Add(1)
			goto re
		}
		rc.readLen += n
		rc.reqTimes = append(rc.reqTimes, int(time.Now().Sub(reqTime).Milliseconds()))
		if atomic.AddInt64(rc.NowNum, 1) >= rc.Count {
			return
		}
	}
}

// Convert bytes to strings
func Bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
