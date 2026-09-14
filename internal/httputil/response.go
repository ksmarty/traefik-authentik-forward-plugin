package httputil

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type ResponseMangler struct {
	http.ResponseWriter
	MangleFunc func(rw http.ResponseWriter)
}

func (rcm *ResponseMangler) WriteHeader(code int) {
	rcm.MangleFunc(rcm.ResponseWriter)
	rcm.ResponseWriter.WriteHeader(code)
}

// Unwrap exposes the wrapped ResponseWriter to http.ResponseController, which
// Traefik uses to upgrade connections.
func (rcm *ResponseMangler) Unwrap() http.ResponseWriter {
	return rcm.ResponseWriter
}

// Hijack forwards the hijack request to the wrapped ResponseWriter. It is
// required for WebSocket and other protocol upgrades: Traefik's reverse proxy
// hijacks the connection to send the 101 response, and fails with
// "not a hijacker" if the middleware's ResponseWriter does not implement
// http.Hijacker.
func (rcm *ResponseMangler) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rcm.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("httputil: underlying ResponseWriter does not support hijacking")
	}

	return hijacker.Hijack()
}

// Flush forwards flushes to the wrapped ResponseWriter, preserving streaming
// behavior through the middleware.
func (rcm *ResponseMangler) Flush() {
	if flusher, ok := rcm.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
