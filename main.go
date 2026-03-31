package main

import (
	"fmt"
	"html/template"

	"vortex/demon"
	"vortex/loadbalancer"

	"github.com/gin-gonic/gin"
)

// func test() {
//     fmt.Println("========================================")
//     fmt.Println(" Vortex: Initializing cluster...")
//     fmt.Println("========================================")

//     // 1. Initial Scale-up (Start all 4 servers here!)
//     demon.AddServers(1)
//     demon.AddServers(3)

//     // 2. THE PAUSE
//     fmt.Println("Waiting for backend nodes to warm up...")
//     // time.Sleep(time.Second)

//     // 3. Setup the Load Balancer Router
//     gin.SetMode(gin.ReleaseMode)
//     r := gin.Default()

//     // 4. The Catch-All Route
//     r.NoRoute(func(c *gin.Context) {
//         loadbalancer.Load_Balancer(c.Writer, c.Request)
//     })

//     fmt.Println("\n Vortex Load Balancer is live on :8000")
//     fmt.Println("========================================")

//	    if err := r.Run(":8000"); err != nil {
//	        fmt.Printf("Vortex failed to start: %v\n", err)
//	    }
//	}
func DashboardHandler(c *gin.Context) {
	const tmpl = `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Vortex Status</title>
		<style>
			body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #0f0f0f; color: #e0e0e0; padding: 40px; }
			.container { max-width: 900px; margin: auto; }
			table { width: 100%; border-collapse: collapse; margin-top: 20px; background: #1a1a1a; border-radius: 8px; overflow: hidden; }
			th { background: #333; color: #888; text-transform: uppercase; font-size: 12px; letter-spacing: 1px; }
			th, td { padding: 15px; text-align: left; border-bottom: 1px solid #2a2a2a; }
			.status-up { color: #4caf50; background: rgba(76, 175, 80, 0.1); padding: 4px 8px; border-radius: 4px; font-size: 12px; }
			.status-down { color: #f44336; background: rgba(244, 67, 54, 0.1); padding: 4px 8px; border-radius: 4px; font-size: 12px; }
			.header { display: flex; justify-content: space-between; align-items: baseline; border-bottom: 2px solid #333; padding-bottom: 10px; }
			.count { color: #00cfd5; font-size: 24px; }
		</style>
		<meta http-equiv="refresh" content="2">
	</head>
	<body>
		<div class="container">
			<div class="header">
				<h1>🌀 Vortex Cluster</h1>
				<div>Active Connections: <span class="count">{{.Active}}</span></div>
			</div>
			<table>
				<thead>
					<tr>
						<th>Backend Node</th>
						<th>Health Status</th>
						<th>Requests Handled</th>
					</tr>
				</thead>
				<tbody>
					{{range .Backends}}
					<tr>
						<td><strong>{{.IP}}</strong></td>
						<td>
							{{if .Healthy}}
								<span class="status-up">● ONLINE</span>
							{{else}}
								<span class="status-down">● OFFLINE</span>
							{{end}}
						</td>
						<td>{{.Count}}</td>
					</tr>
					{{end}}
				</tbody>
			</table>
		</div>
	</body>
	</html>`

	active, servers := loadbalancer.GetStatusData()

	t := template.Must(template.New("vortex").Parse(tmpl))
	t.Execute(c.Writer, struct {
		Active   int64
		Backends []*loadbalancer.Server
	}{
		Active:   active,
		Backends: servers,
	})
}

func main() {
	// test()
	demon.ParseYamlFile()

	go demon.Demon()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.NoRoute(func(c *gin.Context) {
		loadbalancer.Load_Balancer(c.Writer, c.Request)
	})

	fmt.Println("\n Vortex Load Balancer is live on :8000")
	fmt.Println("========================================")

	if err := r.Run(":8000"); err != nil {
		fmt.Printf("Vortex failed to start: %v\n", err)
	}
}
