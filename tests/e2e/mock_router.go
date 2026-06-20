package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	kindclient "github.com/NKS01X/Kind/go-client"
)

func main() {
	log.Println("[Mock Edge Router] Connecting to Kind DB...")
	db, err := kindclient.NewKindClient("localhost:50051")
	if err != nil {
		log.Fatalf("Failed to connect to Kind DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	clientID := "test_client_integration"

	// 1. Send High Traffic Spike
	log.Printf("[Mock Edge Router] Simulating HUGE traffic spike for %s (500 RPS)...", clientID)
	sendMetric(ctx, db, clientID, 500)

	// Wait 30 seconds for daemon to scale up and vman to deploy containers
	log.Println("[Mock Edge Router] Waiting 30s to observe scale up...")
	time.Sleep(30 * time.Second)

	// 2. Send Low Traffic (Scale Down)
	log.Printf("[Mock Edge Router] Traffic dropped for %s (100 RPS)...", clientID)
	sendMetric(ctx, db, clientID, 100)

	// Wait 15 seconds to observe scale down
	log.Println("[Mock Edge Router] Waiting 15s to observe scale down...")
	time.Sleep(15 * time.Second)

	log.Println("[Mock Edge Router] E2E Integration Test Completed!")
}

func sendMetric(ctx context.Context, db *kindclient.KindClient, clientID string, rps int) {
	val, _ := json.Marshal(map[string]interface{}{
		"client_id":   clientID,
		"current_rps": rps,
	})

	// Put into vortex:metrics: to trigger the Watch API in the Daemon
	key := "vortex:metrics:" + clientID
	success, err := db.Put(ctx, key, val)
	if err != nil {
		log.Fatalf("Failed to push metric: %v", err)
	}
	if !success {
		log.Fatalf("Failed to put metric, success=false")
	}
}
