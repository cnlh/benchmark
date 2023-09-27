// Copyright 2020 The benchmark. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Return based on proxy url
func NewProxyConn(proxyUrl string, protocol clientProtocol) (ProxyConn, error) {
	u, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "socks5":
		return &Socks5Client{u, protocol}, nil
	case "http":
		return &HttpClient{u, protocol, nil}, nil
	default:
		return &DefaultClient{}, nil
	}
}

// ProxyConn is used to define the proxy
type ProxyConn interface {
	Dial(network string, address string, timeout time.Duration) (net.Conn, error)
}

// DefaultClient is used to implement a proxy in default
type DefaultClient struct {
	rAddr *net.TCPAddr
}

// Socks5 implementation of ProxyConn
// Set KeepAlive=-1 to reduce the call of syscall
func (dc *DefaultClient) Dial(network string, address string, timeout time.Duration) (conn net.Conn, err error) {
	if dc.rAddr == nil {
		dc.rAddr, err = net.ResolveTCPAddr("tcp", address)
		if err != nil {
			return nil, err
		}
	}
	return net.DialTCP(network, nil, dc.rAddr)
}

type clientProtocol struct {
	transport    string
	quicProtocol string
}

// Socks5Client is used to implement a proxy in socks5
type Socks5Client struct {
	proxyUrl *url.URL
	clientProtocol
}

// Socks5 implementation of ProxyConn
func (s5 *Socks5Client) Dial(network string, address string, timeout time.Duration) (net.Conn, error) {
	var forward proxy.Dialer
	if s5.transport == "quic" {
		forward = NewQuicDialer([]string{s5.quicProtocol})
	}
	d, err := proxy.FromURL(s5.proxyUrl, forward)
	if err != nil {
		return nil, err
	}
	return d.Dial(network, address)
}

// Socks5Client is used to implement a proxy in http
type HttpClient struct {
	proxyUrl *url.URL
	clientProtocol
	qd *QuicDialer
}

func SetHTTPProxyBasicAuth(req *http.Request, username, password string) {
	auth := username + ":" + password
	authEncoded := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Proxy-Authorization", "Basic "+authEncoded)
}

// Http implementation of ProxyConn
func (hc *HttpClient) Dial(network string, address string, timeout time.Duration) (net.Conn, error) {
	req, err := http.NewRequest("CONNECT", "http://"+address, nil)
	if err != nil {
		return nil, err
	}
	password, _ := hc.proxyUrl.User.Password()
	SetHTTPProxyBasicAuth(req, hc.proxyUrl.User.Username(), password)
	var proxyConn net.Conn
	if hc.transport == "quic" {
		if hc.qd == nil {
			hc.qd = NewQuicDialer([]string{hc.quicProtocol})
		}
		proxyConn, err = hc.qd.Dial(network, hc.proxyUrl.Host)
	} else {
		proxyConn, err = net.DialTimeout("tcp", hc.proxyUrl.Host, timeout)
	}
	if err != nil {
		return nil, err
	}
	if err = req.Write(proxyConn); err != nil {
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
