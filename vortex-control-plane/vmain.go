package main

import (
	"context"
	"control-plane/vman"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	kindclient "github.com/NKS01X/Kind/go-client"
)

type ScaleJob struct {
	ClientID string `json:"client_id"`
	Target   int    `json:"target"`
}

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

// Put inserts a key-value pair to Kind DB
func (k *KindClient) Put(ctx context.Context, key string, value []byte) (bool, error) {
	return k.client.Put(ctx, key, value)
}

// Delete removes a key from Kind DB
func (k *KindClient) Delete(ctx context.Context, key string) (bool, error) {
	_, err := k.client.Delete(ctx, key)
	return err == nil, err
}

func main() {
	ctx := context.Background()

	kindWrapper, err := NewKindClient("localhost:50051")
	if err != nil {
		fmt.Printf("Failed to connect to Kind DB: %v\n", err)
		os.Exit(1)
	}
	defer kindWrapper.client.Close()

	kindDB := kindWrapper.client

	manager, err := vman.NewVortexManager()
	if err != nil {
		fmt.Println("Critical Start Error:", err)
		os.Exit(1)
	}

	manager.StartHealthMonitor(ctx, kindWrapper)

	fmt.Println("Manager is running. You can test self-healing by killing a container via 'docker kill <id>' in another terminal.")

	events, err := kindDB.Watch(ctx, "vortex:queue:scale:")
	if err != nil {
		log.Fatalf("Failed to watch queue: %v", err)
	}

	repo := "https://github.com/crccheck/docker-hello-world.git"

	for {
		event, err := events.Recv()
		if err != nil {
			log.Printf("Watch error or closed: %v", err)
			break
		}

		if event.OperationType == "PUT" {
			var job ScaleJob
			if err := json.Unmarshal(event.NewValue, &job); err != nil {
				log.Printf("Failed to parse ScaleJob: %v", err)
				continue
			}

			// Parse client_id from key if needed
			keyParts := strings.Split(event.Key, ":")
			clientID := keyParts[len(keyParts)-1]

			if job.ClientID == "" {
				job.ClientID = clientID
			}

			log.Printf("Processing ScaleJob client=%s target=%d", job.ClientID, job.Target)

			err := manager.Scale(ctx, job.ClientID, repo, job.Target)
			if err != nil {
				log.Printf("Scale Error: %v", err)
			} else {
				log.Printf("ScaleJob ACK \u2713 %s scaled to %d", job.ClientID, job.Target)
			}
		}
	}
}
