package daemon

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	lb "vortex/loadbalancer"
	ratelim "vortex/ratelimiter"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type VortexConfig struct {
	Cluster struct {
		MinReplicas  int `yaml:"min_replicas"`
		MaxReplicas  int `yaml:"max_replicas"`
		StartingPort int `yaml:"starting_port"`
	} `yaml:"cluster"`
	Settings struct {
		Scaleupnumber   int `yaml:"scaleupnumber"`
		Scaledownnumber int `yaml:"scaledownnumber"`
	} `yaml:"settings"`
	RateLimiter struct {
		RateLimit  int           `yaml:"ratelimit"`
		RateWindow time.Duration `yaml:"ratewindow"`
	} `yaml:"ratelimiter"`
}

var (
	config        VortexConfig
	activeServers = make(map[string]*http.Server)
	InitialPort   int
	InitialUrl    = "http://127.0.0.1:"
)

func ParseYamlFile() {
	yamlFile, err := os.ReadFile("vortex.yaml")
	if err != nil {
		fmt.Printf("Fatal error: Could not find vortex.yaml: %v\n", err)
		return
	}

	yaml.Unmarshal(yamlFile, &config)
	InitialPort = config.Cluster.StartingPort
	fmt.Printf("Config loaded! Starting Port: %d, Initial Nodes: %d\n", config.Cluster.StartingPort, config.Cluster.MinReplicas)
	fmt.Println("========================================")
	// SetConfig(config)
	AddServers(config.Cluster.MinReplicas)
}

// this file works with the adding and removing of backends dynamically

// var MaxReplica int
// var MinReplica int
//
// var ServerConfig VortexConfig
// func SetConfig(config VortexConfig) {
//     ServerConfig = config
// }

/*
 * @params = amount of server to spin up
 */
func InsertServer(Url string) {
	lb.AddBackends(Url)
}

func DeleteServer(Url string) {
	lb.RemoveBackends(Url)
}

func StartServer(port int, url string) {
	gin.SetMode(gin.ReleaseMode)
	go ratelim.CleanupStaleVisitors() // cleans the users from the map who's time window has expired
	r := gin.New()
	/*
	 * fetched RateLimit and RateWindow from yaml
	 */
	r.Use(ratelim.CustomRateLimiter(config.RateLimiter.RateLimit, config.RateLimiter.RateWindow))
	r.GET("/", func(c *gin.Context) {
		// for test
		time.Sleep(2 * time.Second)
		c.String(200, "🌀 Vortex Node spinning on Port: %d\n", port)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	activeServers[url] = srv

	fmt.Printf("[Backend] Server started on port %d\n", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("[Backend] Server error on port %d: %v\n", port, err)
	}
}

func AddServers(x int) {
	for i := 0; i < x; i++ {
		Url := fmt.Sprintf("%s%d", InitialUrl, InitialPort+i)
		InsertServer(Url)
		go StartServer(InitialPort+i, Url)
		fmt.Printf("[daemon] Added new backend to pool: %s\n", Url)
	}
	InitialPort += x
	CurrentServers += uint64(x)
}

func RemoveServers(x int) {
	for i := 0; i < x; i++ {
		InitialPort--
		Url := fmt.Sprintf("%s%d", InitialUrl, InitialPort)
		DeleteServer(Url)

		if srv, exists := activeServers[Url]; exists {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(ctx)
			delete(activeServers, Url)
		}

		fmt.Printf("[daemon] Removed backend from pool: %s\n", Url)
	}
	CurrentServers -= uint64(x)
}

var CurrentServers uint64 = 0

func Daemon() {
	for {
		activeReqs := atomic.LoadInt64(&lb.ActiveConnections)

		if uint64(activeReqs) >= uint64(config.Settings.Scaleupnumber) {
			if CurrentServers < uint64(config.Cluster.MaxReplicas) {
				AddServers(1)
			}
		} else if uint64(activeReqs) <= uint64(config.Settings.Scaledownnumber) {
			// RemoveServers(1)
			if CurrentServers > uint64(config.Cluster.MinReplicas) {
				RemoveServers(1)
			}
		}
		time.Sleep(2 * time.Second)
	}
}
