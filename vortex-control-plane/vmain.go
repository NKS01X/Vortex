package main

import (
	"context"
	"control-plane/vman"
	"encoding/json"
	"fmt"
	"os"

	kindclient "github.com/NKS01X/Kind/go-client"
)

// KindClient implements the vman.RouterUpdater interface
type KindClient struct {
	client *kindclient.KindClient
}

func NewKindClient(addr string) (*KindClient, error) {
	c, err := kindclient.NewKindClient(addr)
	if err != nil {
		return nil, err
	}
	return &KindClient{client: c}, nil
}

// UpdateRoutingTable updates the routing table in Kind DB
func (k *KindClient) UpdateRoutingTable(ctx context.Context, clientID string, ips []string) error {
	fmt.Printf("[KIND DB SYNC] Routing updated for %s -> IPs: %v\n", clientID, ips)

	val, err := json.Marshal(map[string]interface{}{
		"client_id": clientID,
		"ips":       ips,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal ips: %w", err)
	}

	_, err = k.client.Put(ctx, "router:"+clientID, val)
	if err != nil {
		return fmt.Errorf("failed to sync to kind db: %w", err)
	}
	return nil
}

func main() {
	ctx := context.Background()

	//Kinddb
	kindDB, err := NewKindClient("localhost:50051")
	if err != nil {
		fmt.Printf("Failed to connect to Kind DB: %v\n", err)
		os.Exit(1)
	}
	defer kindDB.client.Close()

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
