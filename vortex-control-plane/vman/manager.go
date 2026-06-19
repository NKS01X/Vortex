package vman

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// RouterUpdater is an interface that allows vman to talk to KindDB
type RouterUpdater interface {
	UpdateRoutingTable(ctx context.Context, clientID string, ips []string) error
}

// ClientApp holds the desired state for a specific user's deployment
type ClientApp struct {
	ClientID   string
	RepoLink   string
	Containers []string
}

// VortexManager encapsulates the Docker client and the state of all apps
type VortexManager struct {
	docker *client.Client
	apps   map[string]*ClientApp
	mu     sync.RWMutex
}

func NewVortexManager() (*VortexManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to init docker client: %w", err)
	}

	return &VortexManager{
		docker: cli,
		apps:   make(map[string]*ClientApp),
	}, nil
}

func (m *VortexManager) CreateContainer(ctx context.Context, repoLink, clientID string) (string, error) {
	imageTag := fmt.Sprintf("client_%s_%d", clientID, time.Now().UnixNano())

	buildResponse, err := m.docker.ImageBuild(ctx, nil, build.ImageBuildOptions{
		RemoteContext: repoLink,
		Tags:          []string{imageTag},
	})
	if err != nil {
		return "", err
	}
	defer buildResponse.Body.Close()
	_, _ = io.Copy(io.Discard, buildResponse.Body)

	createResp, err := m.docker.ContainerCreate(ctx, &container.Config{
		Image: imageTag,
	}, nil, nil, nil, "")
	if err != nil {
		return "", err
	}

	if err := m.docker.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		return "", err
	}

	return createResp.ID, nil
}

func (m *VortexManager) DeleteContainer(ctx context.Context, containerID string) error {
	return m.docker.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

func (m *VortexManager) Scale(ctx context.Context, clientID, repoLink string, targetReplicas int) error {
	m.mu.Lock()
	app, exists := m.apps[clientID]
	if !exists {
		app = &ClientApp{ClientID: clientID, RepoLink: repoLink, Containers: []string{}}
		m.apps[clientID] = app
	}
	m.mu.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	currentReplicas := len(app.Containers)
	if currentReplicas == targetReplicas {
		return nil
	}

	//scaling down
	if currentReplicas > targetReplicas {
		diff := currentReplicas - targetReplicas
		fmt.Printf("Scaling DOWN %s by %d containers\n", clientID, diff)
		for i := 0; i < diff; i++ {
			lastIdx := len(app.Containers) - 1
			_ = m.DeleteContainer(ctx, app.Containers[lastIdx])
			app.Containers = app.Containers[:lastIdx]
		}
	}
	//scaling up
	if currentReplicas < targetReplicas {
		diff := targetReplicas - currentReplicas
		fmt.Printf("Scaling UP %s by %d containers\n", clientID, diff)
		for i := 0; i < diff; i++ {
			containerID, err := m.CreateContainer(ctx, repoLink, clientID)
			if err != nil {
				return err
			}
			app.Containers = append(app.Containers, containerID)
			fmt.Printf("   [+] Spun up %s\n", containerID[:12])
		}
	}
	return nil
}
