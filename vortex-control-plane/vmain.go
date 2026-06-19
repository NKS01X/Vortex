package vortexcontrolplane

import (
	"context"
	"fmt"
	"os"

	"control-plane/vman"
)

// KindClient implements the vman.RouterUpdater interface
type KindClient struct{}

// UpdateRoutingTable simulates the gRPC call to your Rust database
func (k *KindClient) UpdateRoutingTable(ctx context.Context, clientID string, ips []string) error {
	fmt.Printf("[KIND DB SYNC] Routing updated for %s -> IPs: %v\n", clientID, ips)
	return nil
}

func main() {
	ctx := context.Background()

	//Kinddb
	kindDB := &KindClient{}

	//VortexManager
	manager, err := vman.NewVortexManager()
	if err != nil {
		fmt.Println("Critical Start Error:", err)
		os.Exit(1)
	}

	//health checks
	manager.StartHealthMonitor(ctx, kindDB)

	//Test the deploy logic
	fmt.Println("\n--- Initiating Vortex Deploy ---")
	repo := "https://github.com/docker-library/hello-world.git#master:amd64/hello-world"

	err = manager.Scale(ctx, "user_8x9a2", repo, 2)
	if err != nil {
		fmt.Println("Deploy Error:", err)
	}

	fmt.Println("\nManager is running. You can test self-healing by killing a container via 'docker kill <id>' in another terminal.")

	//blocking for keeping go routines alive
	select {}
}
