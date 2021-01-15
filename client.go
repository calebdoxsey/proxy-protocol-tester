package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pires/go-proxyproto"
	"go.uber.org/zap"
)

func runClient(version byte, dst *url.URL) {
	srcAddr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Fatal("error resolving src", zap.Error(err))
	}

	host := dst.Host
	if !strings.Contains(host, ":") {
		if dst.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	dstAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		log.Fatal("error resolving dst", zap.Error(err))
	}

	hdr := proxyproto.HeaderProxyFromAddrs(version, srcAddr, dstAddr)

	res, err := (&http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := new(net.Dialer).DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}

				_, err = hdr.WriteTo(conn)
				if err != nil {
					conn.Close()
					return nil, err
				}

				return conn, nil
			},
		},
	}).Get(dst.String())
	if err != nil {
		log.Fatal("http error", zap.Error(err))
	}
	fmt.Println("===")
	_, _ = io.Copy(os.Stdout, res.Body)
	fmt.Println("===")
}
