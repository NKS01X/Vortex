package loadbalancer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type Server struct {
	IP string
}

//	var backends = []Server{
//	    {"http://127.0.0.1:3000"},
//	}
var (
	backends          []*Server
	count_request     uint64
	ActiveConnections int64
	mu                sync.RWMutex
)

func AddBackends(url string) {
	mu.Lock()
	defer mu.Unlock()
	backends = append(backends, &Server{IP: url})
}

func RemoveBackends(url string) {
	mu.Lock()
	defer mu.Unlock()
	for i, server := range backends {
		if server.IP == url {
			backends = append(backends[:i], backends[i+1:]...)
			break
		}
	}
}

func getNextBackend() Server {
	mu.RLock()
	defer mu.RUnlock()
	sz_of_backend := len(backends)
	if sz_of_backend == 0 {
		return Server{}
	}
	next := atomic.AddUint64(&count_request, 1)
	return *backends[(uint64(next)-1)%uint64(sz_of_backend)]
}

func Load_Balancer(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&ActiveConnections, 1)
	defer atomic.AddInt64(&ActiveConnections, -1)

	targetUrl := getNextBackend()
	if targetUrl.IP == "" {
		http.Error(w, "No backends available", http.StatusServiceUnavailable)
		return
	}
	fmt.Printf("Routing the reponse to = %s\n", targetUrl.IP)
	target, err := url.Parse(targetUrl.IP)
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	r.Host = target.Host
	proxy.ServeHTTP(w, r)
}

func GetStatusData() (int64, []*Server) {
	mu.RLock()
	defer mu.RUnlock()
	// Return a copy or the slice itself for the template to iterate
	return atomic.LoadInt64(&ActiveConnections), (backends)
}
