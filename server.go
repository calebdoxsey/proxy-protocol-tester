package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/pires/go-proxyproto"
	"go.uber.org/zap"
)

func runServer(addr string) {
	log.Info("starting HTTP server", zap.String("addr", addr))

	li, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("error starting TCP listener", zap.Error(err))
	}
	defer li.Close()

	pli := &proxyListener{li: li}

	srv := &http.Server{
		Handler: http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			hdr := GetHeaderValue(req.Context())
			if hdr == nil {
				res.WriteHeader(http.StatusBadRequest)
				_, _ = res.Write([]byte("No Proxy Protocol Detected"))
				return
			}

			bs, _ := json.MarshalIndent(hdr, "", "  ")
			_, _ = res.Write(bs)
		}),
		ConnContext: pli.ConnContext,
	}
	err = srv.Serve(pli)
	if err != nil {
		log.Fatal("error accepting TCP connection", zap.Error(err))
	}
}

var headerKey = struct{}{}

// GetHeaderValue gets the proxy proto header value for a connection.
func GetHeaderValue(ctx context.Context) *proxyproto.Header {
	hdr, _ := ctx.Value(headerKey).(*proxyproto.Header)
	return hdr
}

// WithHeaderValue sets the proxy proto header value in a context.
func WithHeaderValue(ctx context.Context, hdr *proxyproto.Header) context.Context {
	return context.WithValue(ctx, headerKey, hdr)
}

type proxyListener struct {
	li net.Listener
}

// ConnContext returns the context for a connection
func (li *proxyListener) ConnContext(ctx context.Context, conn net.Conn) context.Context {
	c, ok := conn.(*proxyConn)
	if !ok {
		return ctx
	}
	return WithHeaderValue(ctx, c.hdr)
}

// Accept waits for and returns the next connection to the listener.
func (li *proxyListener) Accept() (net.Conn, error) {
	conn, err := li.li.Accept()
	if err != nil {
		return nil, err
	}

	br := bufio.NewReader(conn)
	hdr, _ := proxyproto.ReadTimeout(br, time.Second*10)

	c := &proxyConn{
		Conn: conn,
		hdr:  hdr,
	}
	if br.Buffered() > 0 {
		c.rdr = io.MultiReader(br, conn)
	} else {
		c.rdr = conn
	}

	return c, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (li *proxyListener) Close() error {
	return li.li.Close()
}

// Addr returns the listener's network address.
func (li *proxyListener) Addr() net.Addr {
	return li.li.Addr()
}

type proxyConn struct {
	net.Conn
	hdr *proxyproto.Header
	rdr io.Reader
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
func (c *proxyConn) Read(b []byte) (n int, err error) {
	return c.rdr.Read(b)
}
