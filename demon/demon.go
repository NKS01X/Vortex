package demon

import (
	"fmt"
	"os"

	lb "vortex/loadbalancer"

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
}

var config VortexConfig

func ParseYamlFile() {
	yamlFile, err := os.ReadFile("vortex.yaml")
	if err != nil {
		fmt.Printf("Fatal error: Could not find vortex.yaml: %v\n", err)
		return
	}

	yaml.Unmarshal(yamlFile, &config)
	fmt.Printf("Config loaded! Starting Port: %d, Initial Nodes: %d\n", config.Cluster.StartingPort, config.Cluster.MinReplicas)
	fmt.Println("========================================")
	// SetConfig(config)
	AddServers(config.Cluster.MinReplicas)
}

// this file works with the adding and removing of backends dynamically
var InitialPort = config.Cluster.StartingPort

var InitialUrl = "http://127.0.0.1:"

// var MaxReplica int
// var MinReplica int
//
// var ServerConfig VortexConfig
// func SetConfig(config VortexConfig) {
// 	ServerConfig = config
// }

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

// func RemoveServers(x int) {
// 	for i := 0; i < x; i++ {
// 	}
// }

var CurrentServers uint64 = 0

func Demon() {
	// r := gin.Default()
	// go r.GET("/", func(c *gin.Context) {
	// 	CurrentServers++
	// 	if(CurrentServers >= uint64(config.Settings.Scaleupnumber)) {
	// 		AddServers(1)
	// 	}else if(CurrentServers <= uint64(config.Settings.Scaledownnumber)) {

	// 	}
	// })
}
