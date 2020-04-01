// Copyright 2020 The benchmark. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"io"
	"strconv"
)

var (
	bodyHeaderSepBytes    = []byte{13, 10, 13, 10}
	bodyHeaderSepBytesLen = 4
	headerSepBytes        = []byte{13, 10}
	contentLengthBytes    = []byte{67, 111, 110, 116, 101, 110, 116, 45, 76, 101, 110, 103, 116, 104, 58, 32}
	contentLengthBytesLen = 16
)

// ConnReadWriter is defines the read and request behaviour of a connection
type ConnReadWriter interface {
	Read(conn io.Reader) (int, error)
	Write(conn io.Writer) (int, error)
}

// HttpReadWriter is a simple and efficient implementation of ConnReadWriter
type HttpReadWriter struct {
	buf        []byte
	writeBytes []byte
}

// Create a new HttpReadWriter
func NewHttpReadWriter(writeBytes []byte) ConnReadWriter {
	return &HttpReadWriter{
		buf:        make([]byte, 65535),
		writeBytes: writeBytes,
	}
}

// Implement the Read func of ConnReadWriter
func (h *HttpReadWriter) Read(r io.Reader) (readLen int, err error) {
	var contentLen string
	var bodyHasRead int
	var headerHasRead int
	var n int
readHeader:
	n, err = r.Read(h.buf[headerHasRead:])
	if err != nil {
		return
	}
	readLen += n
	headerHasRead += n
	var bbArr [2][]byte
	bodyPos := bytes.Index(h.buf[:headerHasRead], bodyHeaderSepBytes)
	if bodyPos > -1 {
		bbArr[0] = h.buf[:bodyPos]
		bbArr[1] = h.buf[bodyPos+bodyHeaderSepBytesLen:]
	} else {
		goto readHeader
	}
	contentStartPos := bytes.Index(bbArr[0], contentLengthBytes)
	start := contentStartPos + contentLengthBytesLen
	end := bytes.Index(bbArr[0][start:], headerSepBytes)
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
		n, err = r.Read(h.buf)
		if err != nil {
			return
		}
		readLen += n
		bodyHasRead += n
	}
	return
}

// Implement the Write func of ConnReadWriter
func (h *HttpReadWriter) Write(r io.Writer) (readLen int, err error) {
	return r.Write(h.writeBytes)
}
