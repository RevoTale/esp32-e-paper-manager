package main

import (
	"crypto/tls"
	"net/http"
	"time"

	"golang.org/x/net/netutil"
)

const maxHTTPSConnections = 16

func httpServer(address string, api http.Handler) *http.Server {
	return &http.Server{Addr: address, Handler: api,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second,
		WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 4096, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13},
		HTTP2: &http.HTTP2Config{MaxConcurrentStreams: 8, MaxReceiveBufferPerConnection: 65536,
			MaxReceiveBufferPerStream: 32768, MaxReadFrameSize: 16384, WriteByteTimeout: 20 * time.Second}}
}

func serveHTTPS(server *http.Server, certificate, key string) error {
	listener, err := listenTCP("tcp", server.Addr)
	if err != nil {
		return err
	}
	listener = netutil.LimitListener(listener, maxHTTPSConnections)
	defer func() { _ = listener.Close() }()
	return server.ServeTLS(listener, certificate, key)
}
