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
	/*
	*
	*	passed as a parameter in CustomRateLimiter
	*
	 */
	// fetch the below 2 from yaml file
	// rateLimit  = 20
	// rateWindow = 10 * time.Second
	//
	//
)

func getHashedIp(ip string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(ip))
	return h.Sum64()
}

func CustomRateLimiter(rateLimit int, rateWindow time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		hashedIp := getHashedIp(ip)

		mu.Lock()
		defer mu.Unlock()

		v, exists := HashedIpToIpnode[hashedIp]

		if !exists || time.Since(v.lastResquestTime) > rateWindow {
			// New visitor OR window expired: Start fresh
			HashedIpToIpnode[hashedIp] = &Ipnode{
				requestcnt:       1,
				lastResquestTime: time.Now(),
			}
			c.Next()
			return
		}

		// If we reached here, the visitor exists and is within the time window
		if int(v.requestcnt) >= rateLimit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "🌀 Vortex Shield: Rate limit exceeded. Back off.",
			})
			return
		}

		// Increment for existing visitor
		v.requestcnt++
		c.Next()
	}
}

func CleanupStaleVisitors() {
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
