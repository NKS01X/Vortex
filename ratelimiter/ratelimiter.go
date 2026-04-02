package ratelimiter

import (
	"hash/fnv"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Ipnode struct {
	requestcnt       uint
	lastResquestTime time.Time
}

// map for hasheip -> Ipnode
var (
	HashedIpToIpnode = make(map[uint64]*Ipnode)
	mu               sync.Mutex
	// fetch the below 2 from yaml file
	rateLimit  = 20
	rateWindow = 10 * time.Second
)

func getHashedIp(ip string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(ip))
	return h.Sum64()
}

func CustomRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		hashedIp := getHashedIp(ip)
		mu.Lock()
		v, exists := HashedIpToIpnode[hashedIp]
		if !exists {
			HashedIpToIpnode[hashedIp] = &Ipnode{requestcnt: 1, lastResquestTime: time.Now()}
			mu.Unlock()
			c.Next()
			return
		}
		// if time expires
		if time.Since(v.lastResquestTime) > rateWindow {
			v.requestcnt = 1
			v.lastResquestTime = time.Now()
			mu.Unlock()
			c.Next()
			return
		} else if int(v.requestcnt) >= rateLimit {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "🌀 Vortex Shield: Rate limit exceeded. Back off.",
			})
			return
		}
		v.requestcnt++
		mu.Unlock()
		c.Next()
	}
}

func cleanupStaleVisitors() {
	for {
		time.Sleep(1 * time.Minute)

		mu.Lock()
		// Loop through the hashes, not the IP strings
		for ipHash, v := range HashedIpToIpnode {
			if time.Since(v.lastResquestTime) > 3*time.Minute {
				delete(HashedIpToIpnode, ipHash)
			}
		}
		mu.Unlock()
	}
}
