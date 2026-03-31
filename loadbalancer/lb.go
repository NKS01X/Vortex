package loadbalancer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	IP      string
	Healthy bool
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
	backends = append(backends, &Server{IP: url, Healthy: true})
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

	var healthy_backends []*Server
	for _, server := range backends {
		if server.Healthy {
			healthy_backends = append(healthy_backends, server)
		}
	}

	sz_of_backend := len(healthy_backends)
	if sz_of_backend == 0 {
		return Server{}
	}
	next := atomic.AddUint64(&count_request, 1)
	return *healthy_backends[(uint64(next)-1)%uint64(sz_of_backend)]
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
	return atomic.LoadInt64(&ActiveConnections), backends
}

func StartHealthCheck() {
	for {
		time.Sleep(10 * time.Second)
		mu.RLock()
		servers := make([]*Server, len(backends))
		copy(servers, backends)
		mu.RUnlock()

		for _, server := range servers {
			go check_health(server)
		}
	}
}

func check_health(server *Server) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(server.IP)

	mu.Lock()
	defer mu.Unlock()

	if err != nil || resp.StatusCode >= 500 {
		server.Healthy = false
	} else {
		server.Healthy = true
	}

	if resp != nil {
		resp.Body.Close()
	}
}
