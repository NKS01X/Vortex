package vman

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaultTestRepo = "https://github.com/nginxinc/docker-nginx.git#master:mainline/alpine"

func init() {
	_ = godotenv.Load("../.env")
}

func getTestRepo() string {
	repo := os.Getenv("TEST_REPO_LINK")
	if repo == "" {
		return defaultTestRepo
	}
	return repo
}

func TestNewVortexManager(t *testing.T) {
	m, err := NewVortexManager()
	require.NoError(t, err)
	require.NotNil(t, m)
	require.NotNil(t, m.docker)
}

func TestCreateAndDeleteContainer(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	clientID := "test_client_create_delete"

	id, err := m.CreateContainer(ctx, repo, clientID)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	err = m.DeleteContainer(ctx, id)
	assert.NoError(t, err)
}

func TestScale(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	clientID := "test_client_scale"

	err = m.Scale(ctx, clientID, repo, 2)
	require.NoError(t, err)

	m.mu.RLock()
	app := m.apps[clientID]
	m.mu.RUnlock()

	require.NotNil(t, app)
	assert.Len(t, app.Containers, 2)

	err = m.Scale(ctx, clientID, repo, 1)
	require.NoError(t, err)

	m.mu.RLock()
	assert.Len(t, app.Containers, 1)
	m.mu.RUnlock()

	err = m.Scale(ctx, clientID, repo, 0)
	require.NoError(t, err)

	m.mu.RLock()
	assert.Len(t, app.Containers, 0)
	m.mu.RUnlock()
}

func TestScaleRace(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	clientID := "test_client_race"

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(target int) {
			defer wg.Done()
			_ = m.Scale(ctx, clientID, repo, target)
		}(i % 3)
	}

	wg.Wait()

	_ = m.Scale(ctx, clientID, repo, 0)
}

func TestScaleNegativeTarget(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	clientID := "test_client_negative"

	err = m.Scale(ctx, clientID, repo, -1)
	require.NoError(t, err)

	m.mu.RLock()
	app := m.apps[clientID]
	m.mu.RUnlock()

	require.NotNil(t, app)
	assert.Len(t, app.Containers, 0)

	_ = m.Scale(ctx, clientID, repo, 0)
}

func TestScaleConcurrentSameTarget(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	clientID := "test_client_concurrent_same"

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.Scale(ctx, clientID, repo, 2)
		}()
	}

	wg.Wait()

	m.mu.RLock()
	app := m.apps[clientID]
	m.mu.RUnlock()

	require.NotNil(t, app)
	assert.Len(t, app.Containers, 2)

	_ = m.Scale(ctx, clientID, repo, 0)
}

func TestDeleteContainerNotFound(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	err = m.DeleteContainer(ctx, "nonexistent-container-id")
	assert.Error(t, err)
}

func TestCreateContainerInvalidRepo(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	_, err = m.CreateContainer(ctx, "invalid/repo/that/does/not/exist", "test_client")
	assert.Error(t, err)
}

func TestNewVortexManagerRecoversState(t *testing.T) {
	m1, err := NewVortexManager()
	require.NoError(t, err)

	ctx := context.Background()
	repo := getTestRepo()
	clientID := "test_recovery_client"

	err = m1.Scale(ctx, clientID, repo, 1)
	require.NoError(t, err)

	m1.mu.RLock()
	containerCount := len(m1.apps[clientID].Containers)
	m1.mu.RUnlock()
	require.Equal(t, 1, containerCount)

	m2, err := NewVortexManager()
	require.NoError(t, err)

	m2.mu.RLock()
	app := m2.apps[clientID]
	m2.mu.RUnlock()

	require.NotNil(t, app)
	assert.Len(t, app.Containers, 1)

	_ = m2.Scale(ctx, clientID, repo, 0)
}

func TestScaleZeroToZero(t *testing.T) {
	ctx := context.Background()
	m, err := NewVortexManager()
	require.NoError(t, err)

	repo := getTestRepo()
	clientID := "test_client_zero_zero"

	err = m.Scale(ctx, clientID, repo, 0)
	require.NoError(t, err)

	m.mu.RLock()
	app, exists := m.apps[clientID]
	m.mu.RUnlock()

	assert.True(t, exists)
	assert.Len(t, app.Containers, 0)
}
