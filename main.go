package main

import (
	"fmt"
	"os"

	"vortex/demon"
	"vortex/loadbalancer"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type VortexConfig struct {
	Cluster struct {
		MinReplicas  int `yaml:"min_replicas"`
		MaxReplicas  int `yaml:"max_replicas"`
		StartingPort int `yaml:"starting_port"`
	} `yaml:"cluster"`
}

func ParseYamlFile() {
	yamlFile, err := os.ReadFile("vortex.yaml")
	if err != nil {
		fmt.Printf("Fatal error: Could not find vortex.yaml: %v\n", err)
		return
	}
	var config VortexConfig
	yaml.Unmarshal(yamlFile, &config)
	fmt.Printf("Config loaded! Starting Port: %d, Initial Nodes: %d\n", config.Cluster.StartingPort, config.Cluster.MinReplicas)
	fmt.Println("========================================")
	demon.SetConfig(config.Cluster.StartingPort)
	demon.AddServers(config.Cluster.MinReplicas)
}

func test() {
	fmt.Println("========================================")
	fmt.Println(" Vortex: Initializing cluster...")
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
	ParseYamlFile()
}
