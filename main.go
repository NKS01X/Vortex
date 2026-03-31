package main

import (
	"fmt"

	"vortex/demon"
	"vortex/loadbalancer"

	"github.com/gin-gonic/gin"
)

func test() {
	fmt.Println("========================================")
	fmt.Println("🌀 Vortex: Initializing cluster...")
	fmt.Println("========================================")

	// 1. Initial Scale-up (Start all 4 servers here!)
	demon.AddServers(1)
	demon.AddServers(3)

	// 2. THE PAUSE
	fmt.Println("Waiting for backend nodes to warm up...")
	// time.Sleep(time.Second)

	// 3. Setup the Load Balancer Router
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 4. The Catch-All Route
	r.NoRoute(func(c *gin.Context) {
		loadbalancer.Load_Balancer(c.Writer, c.Request)
	})

	fmt.Println("\n Vortex Load Balancer is live on :8000")
	fmt.Println("========================================")

	if err := r.Run(":8000"); err != nil {
		fmt.Printf("Vortex failed to start: %v\n", err)
	}
}

func main() {
	// test()
}
