// Copyright 2020 The benchmark. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"errors"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Return based on proxy url
func NewProxyConn(proxyUrl string) (ProxyConn, error) {
	u, err := url.Parse(proxyUrl)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "socks5":
		return &Socks5Client{u}, nil
	case "http":
		return &HttpClient{u}, nil
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

// Socks5Client is used to implement a proxy in socks5
type Socks5Client struct {
	proxyUrl *url.URL
}

// Socks5 implementation of ProxyConn
func (s5 *Socks5Client) Dial(network string, address string, timeout time.Duration) (net.Conn, error) {
	d, err := proxy.FromURL(s5.proxyUrl, nil)
	if err != nil {
		return nil, err
	}
	return d.Dial(network, address)
}

// Socks5Client is used to implement a proxy in http
type HttpClient struct {
	proxyUrl *url.URL
}

// Http implementation of ProxyConn
func (hc *HttpClient) Dial(network string, address string, timeout time.Duration) (net.Conn, error) {
	req, err := http.NewRequest("CONNECT", "http://"+address, nil)
	if err != nil {
		return nil, err
	}
	password, _ := hc.proxyUrl.User.Password()
	req.SetBasicAuth(hc.proxyUrl.User.Username(), password)
	proxyConn, err := net.DialTimeout("tcp", hc.proxyUrl.Host, timeout)
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
