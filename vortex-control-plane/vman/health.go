package vman

import (
	"context"
	"fmt"
	"time"
)

// StartHealthMonitor starts the background reconciliation loop.
func (m *VortexManager) StartHealthMonitor(ctx context.Context, db RouterUpdater) {
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
func (m *VortexManager) runHealthCheckCycle(ctx context.Context, db RouterUpdater) {
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

				// SELF-HEAL
				fmt.Printf("Self-Healing: Spinning up replacement for %s...\n", clientID)
				newID, err := m.CreateContainer(ctx, app.RepoLink, clientID)

				if err == nil {
					healthyContainers = append(healthyContainers, newID)
					newInfo, _ := m.docker.ContainerInspect(ctx, newID)
					healthyIPs = append(healthyIPs, newInfo.NetworkSettings.IPAddress)
				} else {
					fmt.Printf("Failed to self-heal %s: %v\n", clientID, err)
				}
			} else {
				// HEALTHY CONTAINER
				healthyContainers = append(healthyContainers, containerID)
				healthyIPs = append(healthyIPs, inspectInfo.NetworkSettings.IPAddress)
			}
		}

		app.Containers = healthyContainers

		// Push updates to Kind DB
		_ = db.UpdateRoutingTable(ctx, clientID, healthyIPs)
	}
}
