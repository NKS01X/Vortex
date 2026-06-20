package main

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockKindClient struct {
	mu          sync.Mutex
	casResults  map[string]bool
	watchEvents map[string][]WatchEvent
	puts        map[string][]byte
	closed      bool
}

func NewMockKindClient() *MockKindClient {
	return &MockKindClient{
		casResults:  make(map[string]bool),
		watchEvents: make(map[string][]WatchEvent),
		puts:        make(map[string][]byte),
	}
}

func (m *MockKindClient) SetCASResult(key string, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.casResults[key] = success
}

func (m *MockKindClient) AddWatchEvents(prefix string, events []WatchEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watchEvents[prefix] = events
}

func (m *MockKindClient) CAS(ctx context.Context, key string, oldValue, newValue []byte, ttlMs int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if success, ok := m.casResults[key]; ok {
		return success, nil
	}
	return true, nil
}

func (m *MockKindClient) Watch(ctx context.Context, prefix string) (WatchStream, error) {
	m.mu.Lock()
	events := m.watchEvents[prefix]
	m.mu.Unlock()

	return &MockWatchStream{events: events}, nil
}

func (m *MockKindClient) Put(ctx context.Context, key string, value []byte) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.puts[key] = value
	return true, nil
}

func (m *MockKindClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

type MockWatchStream struct {
	events []WatchEvent
	idx    int
}

func (m *MockWatchStream) Recv() (*WatchEvent, error) {
	if m.idx >= len(m.events) {
		return nil, context.Canceled
	}
	evt := m.events[m.idx]
	m.idx++
	return &evt, nil
}

func TestMetricsUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Metrics
		wantErr bool
	}{
		{
			name:  "valid metrics",
			input: `{"client_id": "client-1", "current_rps": 150}`,
			want:  Metrics{ClientID: "client-1", CurrentRPS: 150},
		},
		{
			name:  "zero rps",
			input: `{"client_id": "client-2", "current_rps": 0}`,
			want:  Metrics{ClientID: "client-2", CurrentRPS: 0},
		},
		{
			name:    "invalid json",
			input:   `{"client_id": "client-3", "current_rps": }`,
			wantErr: true,
		},
		{
			name:    "missing fields",
			input:   `{}`,
			want:    Metrics{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Metrics
			err := json.Unmarshal([]byte(tt.input), &m)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, m)
			}
		})
	}
}

func TestScaleJobMarshal(t *testing.T) {
	job := ScaleJob{ClientID: "client-1", Target: 3}
	data, err := json.Marshal(job)
	require.NoError(t, err)

	var decoded ScaleJob
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, job, decoded)
}

func TestTargetReplicasCalculation(t *testing.T) {
	tests := []struct {
		name     string
		rps      int
		expected int
	}{
		{"zero rps", 0, 0},
		{"low rps", 50, 1},
		{"exact 100", 100, 1},
		{"200 rps", 200, 2},
		{"350 rps", 350, 3},
		{"1000 rps", 1000, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.rps / 100
			if target == 0 && tt.rps > 0 {
				target = 1
			}
			assert.Equal(t, tt.expected, target)
		})
	}
}

func TestClientCacheOperations(t *testing.T) {
	cache := make(map[string]int)

	cache["client-1"] = 2
	cache["client-2"] = 3

	total := 0
	for _, v := range cache {
		total += v
	}
	assert.Equal(t, 5, total)

	cache["client-1"] = 5
	total = 0
	for _, v := range cache {
		total += v
	}
	assert.Equal(t, 8, total)
}

func TestPrometheusMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_active_nodes",
		Help: "Test gauge",
	})
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_requests_total",
		Help: "Test counter",
	}, []string{"status"})

	reg.MustRegister(gauge, counter)

	gauge.Set(5)
	counter.WithLabelValues("200").Add(100)
	counter.WithLabelValues("429").Add(5)

	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	var gaugeVal, counter200, counter429 float64
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_active_nodes" {
			gaugeVal = mf.GetMetric()[0].GetGauge().GetValue()
		}
		if mf.GetName() == "test_requests_total" {
			for _, m := range mf.GetMetric() {
				if m.GetLabel()[0].GetValue() == "200" {
					counter200 = m.GetCounter().GetValue()
				}
				if m.GetLabel()[0].GetValue() == "429" {
					counter429 = m.GetCounter().GetValue()
				}
			}
		}
	}

	assert.Equal(t, 5.0, gaugeVal)
	assert.Equal(t, 100.0, counter200)
	assert.Equal(t, 5.0, counter429)
}

func TestRenewLock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mockDB := NewMockKindClient()
	mockDB.SetCASResult("vortex:daemon:leader", true)

	done := make(chan struct{})
	go func() {
		renewLock(ctx, mockDB)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("context cancellation did not stop goroutine")
	}
}

func TestWatchStreamProcessing(t *testing.T) {
	mockDB := NewMockKindClient()

	events := []WatchEvent{
		{OperationType: "PUT", NewValue: []byte(`{"client_id": "c1", "current_rps": 250}`)},
		{OperationType: "PUT", NewValue: []byte(`{"client_id": "c2", "current_rps": 50}`)},
	}
	mockDB.AddWatchEvents("vortex:metrics:", events)

	ClientCache = make(map[string]int)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	RunWatchStream(ctx, mockDB)

	mockDB.mu.Lock()
	puts := mockDB.puts
	mockDB.mu.Unlock()

	assert.Contains(t, puts, "vortex:queue:scale:c1")
	assert.Contains(t, puts, "vortex:queue:scale:c2")

	var job1 ScaleJob
	err := json.Unmarshal(puts["vortex:queue:scale:c1"], &job1)
	require.NoError(t, err)
	assert.Equal(t, "c1", job1.ClientID)
	assert.Equal(t, 2, job1.Target)

	var job2 ScaleJob
	err = json.Unmarshal(puts["vortex:queue:scale:c2"], &job2)
	require.NoError(t, err)
	assert.Equal(t, "c2", job2.ClientID)
	assert.Equal(t, 1, job2.Target)
}
