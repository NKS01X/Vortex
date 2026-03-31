package demon

import (
	"fmt"

	lb "vortex/loadbalancer"

	"github.com/gin-gonic/gin"
)

// this file works with the adding and removing of backends dynamically

var (
	InitialPort = 3001
	InitialUrl  = "http://127.0.0.1:"
)

/*
 * @params = amount of server to spin up
 */
func InsertServer(Url string) {
	lb.AddBackends(Url)
}

func StartServer(port int) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		c.String(200, "🌀 Vortex Node spinning on Port: %d\n", port)
	})
	fmt.Printf("[Backend] Server started on port %d\n", port)
	r.Run(fmt.Sprintf(":%d", port))
}

func AddServers(x int) {
	for i := 0; i < x; i++ {
		Url := fmt.Sprintf("%s%d", InitialUrl, InitialPort+i)
		InsertServer(Url)
		go StartServer(InitialPort + i)
		fmt.Printf("[Demon] Added new backend to pool: %s\n", Url)
	}
	InitialPort += x
}

func Demon() {
}
