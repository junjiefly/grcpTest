package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var ClientTimeout *http.Client
var TransportTimeout *http.Transport

var ServerOffLineError = errors.New("ServerIsOffLine")

func init() {
	TransportTimeout = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   30 * time.Second,
		MaxIdleConnsPerHost:   16,
		ResponseHeaderTimeout: time.Second * 10,
	}
	ClientTimeout = &http.Client{Transport: TransportTimeout, Timeout: 10 * time.Second}
}

// Conn wraps a net.Conn, and sets a deadline for every read
// and write operation.
type Conn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Listener struct {
	net.Listener
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	stopped      *bool
}

func (c *Conn) Read(b []byte) (count int, e error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	count, e = c.Conn.Read(b)
	return
}

func (c *Conn) Write(b []byte) (count int, e error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	count, e = c.Conn.Write(b)
	return
}

func (c *Conn) Close() error {
	return c.Conn.Close()
}

func NewListener(addr string, timeout time.Duration) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	tl := &Listener{
		Listener:     l,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}
	return tl, nil
}
func (l *Listener) Accept() (net.Conn, error) {
	if l.stopped != nil && *l.stopped {
		return nil, ServerOffLineError
	}
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	tc := &Conn{
		Conn:         c,
		ReadTimeout:  l.ReadTimeout,
		WriteTimeout: l.WriteTimeout,
	}
	return tc, nil
}

type CounterReader struct {
	reader    io.Reader
	bytesRead int
}

func (r *CounterReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.bytesRead += n
	return n, err
}

func CloseResp(resp *http.Response) {
	if resp == nil {
		return
	}
	r := &CounterReader{reader: resp.Body}
	io.Copy(io.Discard, r)
	resp.Body.Close()
	if r.bytesRead > 0 {
		fmt.Println("response leftover", r.bytesRead, "bytes")
	}
}
