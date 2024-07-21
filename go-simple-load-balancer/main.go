package main

import (
	"net/http"
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

// Avoid race conditions
func (s *Server) SetAlive(alive bool) {
	s.mux.Lock()
	s.Alive = alive
	s.mux.Unlock()
}

func (s *Server) isAlive() (alive bool) {
	s.mux.RLock()
	alive = s.Alive
	s.mux.RUnlock()
	return
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.servers)))
}

// returns next active peer to take a connection
func (s *ServerPool) GetNextPeer() *Server {
	// Loop entire backend to search for alive server
	next := s.NextIndex()
	l := len(s.servers) + next // start from next index and move full cycle

	for i := next; i < l; i++ {
		idx := i % len(s.servers) // take an index by modding with length

		if s.servers[idx].isAlive() {
			if i != next { // original one

				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.servers[idx]
		}
	}
	return nil
}

func main() {
	u, _ := url.Parse("http://localhost:8080")

	rp := httputil.NewSingleHostReverseProxy(u)

	http.HandlerFunc(rp.ServeHTTP)
}
