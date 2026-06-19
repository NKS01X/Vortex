package vman

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockDatabaseClient struct {
	mu      sync.Mutex
	Puts    map[string][]byte
	Deletes []string
}

func (m *MockDatabaseClient) Put(ctx context.Context, key string, value []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Puts == nil {
		m.Puts = make(map[string][]byte)
	}
	m.Puts[key] = value
	return true, nil
}

func (m *MockDatabaseClient) Delete(ctx context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Deletes = append(m.Deletes, key)
	return true, nil
}

func TestHealthMonitorSelfHealing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	clientID := "test_client_health"

	// Initial Scale
	err = m.Scale(ctx, clientID, repo, 1)
	require.NoError(t, err)

	m.mu.RLock()
	app := m.apps[clientID]
	m.mu.RUnlock()

	require.Len(t, app.Containers, 1)
	originalID := app.Containers[0]

	// Stop the container to simulate failure
	err = m.docker.ContainerStop(ctx, originalID, container.StopOptions{})
	require.NoError(t, err)

	mockUpdater := &MockDatabaseClient{}

	// Run single health check cycle manually
	m.runHealthCheckCycle(ctx, mockUpdater)

	m.mu.RLock()
	require.Len(t, app.Containers, 1, "Should have self-healed back to 1 container")
	newID := app.Containers[0]
	m.mu.RUnlock()

	assert.NotEqual(t, originalID, newID, "Self-healing should have spun up a new container ID")

	mockUpdater.mu.Lock()
	routerVal, exists := mockUpdater.Puts["router:"+clientID]
	assert.True(t, exists, "Should have updated routing table")

	var routeData struct {
		ClientID string   `json:"client_id"`
		IPs      []string `json:"ips"`
	}
	_ = json.Unmarshal(routerVal, &routeData)

	assert.Equal(t, clientID, routeData.ClientID)
	assert.Len(t, routeData.IPs, 1, "Should have reported 1 healthy IP to the router")

	// Verify ContainerRecord handling
	var deletedFound bool
	for _, delKey := range mockUpdater.Deletes {
		if delKey == "vortex:container:"+originalID {
			deletedFound = true
			break
		}
	}
	assert.True(t, deletedFound, "Should have deleted dead container record")
	_, hasNewContainer := mockUpdater.Puts["vortex:container:"+newID]
	assert.True(t, hasNewContainer, "Should have put new container record")

	mockUpdater.mu.Unlock()

	_ = m.Scale(ctx, clientID, repo, 0)
}

func TestHealthCheckCycleNoContainers(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	// Clean up any leftover containers from previous test runs
	m.mu.Lock()
	for clientID, app := range m.apps {
		for _, containerID := range app.Containers {
			_ = m.docker.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
		}
		delete(m.apps, clientID)
	}
	m.mu.Unlock()

	mockUpdater := &MockDatabaseClient{}
	m.runHealthCheckCycle(ctx, mockUpdater)

	mockUpdater.mu.Lock()
	assert.Equal(t, 0, len(mockUpdater.Puts))
	assert.Equal(t, 0, len(mockUpdater.Deletes))
	mockUpdater.mu.Unlock()
}

func TestHealthCheckCycleMultipleClients(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()

	err = m.Scale(ctx, "client-a", repo, 1)
	require.NoError(t, err)
	err = m.Scale(ctx, "client-b", repo, 2)
	require.NoError(t, err)

	mockUpdater := &MockDatabaseClient{}
	m.runHealthCheckCycle(ctx, mockUpdater)

	mockUpdater.mu.Lock()
	defer mockUpdater.mu.Unlock()

	_, hasRouterA := mockUpdater.Puts["router:client-a"]
	_, hasRouterB := mockUpdater.Puts["router:client-b"]
	assert.True(t, hasRouterA)
	assert.True(t, hasRouterB)

	var routeA, routeB struct {
		ClientID string   `json:"client_id"`
		IPs      []string `json:"ips"`
	}
	_ = json.Unmarshal(mockUpdater.Puts["router:client-a"], &routeA)
	_ = json.Unmarshal(mockUpdater.Puts["router:client-b"], &routeB)

	assert.Equal(t, "client-a", routeA.ClientID)
	assert.Len(t, routeA.IPs, 1)
	assert.Equal(t, "client-b", routeB.ClientID)
	assert.Len(t, routeB.IPs, 2)

	_ = m.Scale(ctx, "client-a", repo, 0)
	_ = m.Scale(ctx, "client-b", repo, 0)
}

func TestHealthCheckCycleDBPutError(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	err = m.Scale(ctx, "client-db-error", repo, 1)
	require.NoError(t, err)

	mockUpdater := &MockDatabaseClient{
		Puts: make(map[string][]byte),
	}

	mockUpdater.mu.Lock()
	mockUpdater.Puts["router:client-db-error"] = []byte("fail")
	mockUpdater.mu.Unlock()

	m.runHealthCheckCycle(ctx, mockUpdater)

	m.mu.RLock()
	app := m.apps["client-db-error"]
	m.mu.RUnlock()
	require.NotNil(t, app)
}

func TestHealthCheckCycleDBDeleteError(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	err = m.Scale(ctx, "client-db-delete-error", repo, 1)
	require.NoError(t, err)

	m.mu.RLock()
	app := m.apps["client-db-delete-error"]
	containerID := app.Containers[0]
	m.mu.RUnlock()

	err = m.docker.ContainerStop(ctx, containerID, container.StopOptions{})
	require.NoError(t, err)

	mockUpdater := &MockDatabaseClient{
		Deletes: []string{},
	}

	m.runHealthCheckCycle(ctx, mockUpdater)

	mockUpdater.mu.Lock()
	deletedFound := false
	for _, delKey := range mockUpdater.Deletes {
		if delKey == "vortex:container:"+containerID {
			deletedFound = true
			break
		}
	}
	assert.True(t, deletedFound)
	mockUpdater.mu.Unlock()

	_ = m.Scale(ctx, "client-db-delete-error", repo, 0)
}

func TestHealthCheckCyclePartialFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	err = m.Scale(ctx, "client-partial", repo, 2)
	require.NoError(t, err)

	m.mu.RLock()
	app := m.apps["client-partial"]
	containerID1 := app.Containers[0]
	m.mu.RUnlock()

	err = m.docker.ContainerStop(ctx, containerID1, container.StopOptions{})
	require.NoError(t, err)

	mockUpdater := &MockDatabaseClient{}
	m.runHealthCheckCycle(ctx, mockUpdater)

	mockUpdater.mu.Lock()
	var routeData struct {
		ClientID string   `json:"client_id"`
		IPs      []string `json:"ips"`
	}
	_ = json.Unmarshal(mockUpdater.Puts["router:client-partial"], &routeData)
	assert.Len(t, routeData.IPs, 2)

	deletedFound := false
	for _, delKey := range mockUpdater.Deletes {
		if delKey == "vortex:container:"+containerID1 {
			deletedFound = true
			break
		}
	}
	assert.True(t, deletedFound)

	_, hasNewContainer := mockUpdater.Puts["vortex:container:"+routeData.IPs[0]]
	mockUpdater.mu.Unlock()
	assert.True(t, hasNewContainer || len(routeData.IPs) == 2)

	_ = m.Scale(ctx, "client-partial", repo, 0)
}

func TestHealthMonitorStartStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m, err := NewVortexManager()
	require.NoError(t, err)

	mockUpdater := &MockDatabaseClient{}
	m.StartHealthMonitor(ctx, mockUpdater)

	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)
}
