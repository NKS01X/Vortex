package loadbalancer

import "sync/atomic"

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
