package demon

import (
	"fmt"

	lb "vortex/loadbalancer"
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

func AddServers(x int) {
	for i := 0; i < x; i++ {
		Url := fmt.Sprintf("%s%d", InitialUrl, InitialPort+i)
		InsertServer(Url)
		fmt.Printf("[Demon] Added new backend to pool: %s\n", Url)
	}
	InitialPort += x
}

func Demon() {
}
