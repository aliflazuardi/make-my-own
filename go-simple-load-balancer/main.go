package main

import (
	"context"
	"log"
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

var serverPool ServerPool

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

func (s *ServerPool) MarkServerStatus(url string, status bool) {
}

func lb(w http.ResponseWriter, r *http.Request) {
	peer := serverPool.GetNextPeer()

	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func main() {
	u, _ := url.Parse("http://localhost:8080")

	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
		log.Printf("[%s] %s\n", "host", e.Error())

		serverPool.MarkServerStatus(serverUrl, false)

		ctx := context.WithValue(request.Context(), Attempts, attemps+1)
		lb(writer, request.WithContext(ctx))
	}
}
