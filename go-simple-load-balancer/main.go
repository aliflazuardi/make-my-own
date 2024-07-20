package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type Server struct {
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy
	mux          sync.RWMutex
	Alive        bool
}

type ServerPool struct {
	servers []*Server
	current uint64
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.servers)))
}

func main() {
}
