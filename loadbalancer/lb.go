package loadbalancer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type servers struct {
	IP string
}

var backend []servers

var count_request uint64

func getNextBackend() servers {
	sz_of_backend := len(backend)
	next := atomic.AddUint64(&count_request, 1)
	return backend[(uint64(next)-1)%uint64(sz_of_backend)]
}

func Load_Balancer(w http.ResponseWriter, r *http.Request) {
	targetUrl := getNextBackend()
	fmt.Printf("Routing the reponse to = %s\n", targetUrl)
	target, err := url.Parse(targetUrl.IP)
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	r.Host = target.Host
	proxy.ServeHTTP(w, r)
}
