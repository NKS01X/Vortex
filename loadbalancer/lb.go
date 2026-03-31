package loadbalancer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type Server struct {
	IP string
}

//	var backends = []Server{
//		{"http://127.0.0.1:3000"},
//	}
var (
	backends      []Server
	count_request uint64
)

func AddBackends(url string) {
	backends = append(backends, Server{IP: url})
}

func getNextBackend() Server {
	sz_of_backend := len(backends)
	next := atomic.AddUint64(&count_request, 1)
	return backends[(uint64(next)-1)%uint64(sz_of_backend)]
}

func Load_Balancer(w http.ResponseWriter, r *http.Request) {
	targetUrl := getNextBackend()
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
