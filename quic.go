package main

import (
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"golang.org/x/net/context"
	"net"
	"sync"
	"sync/atomic"
)

type QuicDialer struct {
	NextProtos []string
	streams    atomic.Uint32
	c          quic.Connection
	dialing    sync.RWMutex
}

func NewQuicDialer(nextProtos []string) *QuicDialer {
	return &QuicDialer{
		NextProtos: nextProtos,
	}
}

const maxStreams = 32

func (d *QuicDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	now := d.streams.Add(1)
	if now > maxStreams {
		// wait for dialing
		d.dialing.RLock()
		d.dialing.RUnlock()
		return d.DialContext(ctx, network, address)
	}
	if now == maxStreams+1 || now == 1 {
		d.dialing.Lock()
		c, err := quic.DialAddr(ctx, address, &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         d.NextProtos,
		}, nil)
		if err != nil {
			d.dialing.Unlock()
			return nil, err
		}
		d.c = c
		d.streams.Store(1)
		d.dialing.Unlock()
	}
	if d.c == nil {
		// still in initial dialing
		return d.DialContext(ctx, network, address)
	}
	s, err := d.c.OpenStreamSync(ctx)
	if err != nil {
		// dial a new connection in next time
		d.streams.Store(0)
		return nil, err
	}
	return NewStream(s, d.c.LocalAddr(), d.c.RemoteAddr()), nil
}

func (d *QuicDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

type Stream struct {
	quic.Stream
	lAddr net.Addr
	rAddr net.Addr
}

func NewStream(s quic.Stream, lAddr, rAddr net.Addr) net.Conn {
	return &Stream{
		Stream: s,
		lAddr:  lAddr,
		rAddr:  rAddr,
	}
}

func (s *Stream) LocalAddr() net.Addr {
	return s.lAddr
}

func (s *Stream) RemoteAddr() net.Addr {
	return s.rAddr
}

func (s *Stream) Close() error {
	s.CancelRead(0)
	return s.Stream.Close()
}
