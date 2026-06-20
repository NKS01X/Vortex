package vman

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// StartHealthMonitor starts the background reconciliation loop.
func (m *VortexManager) StartHealthMonitor(ctx context.Context, db DatabaseClient) {
	fmt.Println("Starting Vortex Health Monitor (5s ticks)...")
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Shutting down health monitor")
				ticker.Stop()
				return
			case <-ticker.C:
				m.runHealthCheckCycle(ctx, db)
			}
		}
	}()
}

// runHealthCheckCycle inspects all containers and self-heals any dead ones.
func (m *VortexManager) runHealthCheckCycle(ctx context.Context, db DatabaseClient) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for clientID, app := range m.apps {
		var healthyContainers []string
		var healthyIPs []string

		for _, containerID := range app.Containers {
			inspectInfo, err := m.docker.ContainerInspect(ctx, containerID)

			if err != nil || !inspectInfo.State.Running {
				// DEAD CONTAINER DETECTED
				status := "missing"
				if err == nil {
					status = inspectInfo.State.Status
				}
				fmt.Printf("ALERT: Container %s for %s is DEAD (Status: %s)\n", containerID[:12], clientID, status)

				_ = m.DeleteContainer(ctx, containerID)
				_, _ = db.Delete(ctx, "vortex:container:"+containerID)

				// SELF-HEAL
				fmt.Printf("Self-Healing: Spinning up replacement for %s...\n", clientID)
				newID, err := m.CreateContainer(ctx, app.RepoLink, clientID)

				if err == nil {
					healthyContainers = append(healthyContainers, newID)
					newInfo, _ := m.docker.ContainerInspect(ctx, newID)
					healthyIPs = append(healthyIPs, newInfo.NetworkSettings.IPAddress)

					// Sync ContainerRecord for new container
					crVal, _ := json.Marshal(map[string]interface{}{
						"id":         newID,
						"client_id":  clientID,
						"status":     "Running",
						"ip_address": newInfo.NetworkSettings.IPAddress,
						"repo_link":  app.RepoLink,
					})
					_, _ = db.Put(ctx, "vortex:container:"+newID, crVal)
				} else {
					fmt.Printf("Failed to self-heal %s: %v\n", clientID, err)
				}
			} else {
				// HEALTHY CONTAINER
				healthyContainers = append(healthyContainers, containerID)
				healthyIPs = append(healthyIPs, inspectInfo.NetworkSettings.IPAddress)

				// Sync ContainerRecord
				crVal, _ := json.Marshal(map[string]interface{}{
					"id":         containerID,
					"client_id":  clientID,
					"status":     "Running",
					"ip_address": inspectInfo.NetworkSettings.IPAddress,
					"repo_link":  app.RepoLink,
				})
				_, _ = db.Put(ctx, "vortex:container:"+containerID, crVal)
			}
		}

		app.Containers = healthyContainers

		// Push updates to Kind DB
		val, _ := json.Marshal(map[string]interface{}{
			"client_id": clientID,
			"ips":       healthyIPs,
		})
		_, _ = db.Put(ctx, "router:"+clientID, val)
	}
}
