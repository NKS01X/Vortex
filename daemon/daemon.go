package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	ClientID   string `json:"client_id"`
	CurrentRPS int    `json:"current_rps"`
}

type ScaleJob struct {
	ClientID string `json:"client_id"`
	Target   int    `json:"target"`
}

type KindClient interface {
	CAS(ctx context.Context, key string, oldValue, newValue []byte, ttlMs int) (bool, error)
	Watch(ctx context.Context, prefix string) (WatchStream, error)
	Put(ctx context.Context, key string, value []byte) (bool, error)
	Close() error
}

type WatchStream interface {
	Recv() (*WatchEvent, error)
}

type WatchEvent struct {
	OperationType string
	NewValue      []byte
}

var (
	ClientCache = make(map[string]int)

	ActiveNodesGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "vortex_active_nodes",
		Help: "The total number of active Vortex replica nodes",
	})
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vortex_requests_total",
		Help: "The total number of processed requests",
	}, []string{"status"})
)

func main() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Prometheus metrics listening on :8000/metrics")
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	db, err := NewKindClient("localhost:50051")
	if err != nil {
		log.Fatalf("Failed to connect to Kind DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	for {
		success, err := db.CAS(ctx, "vortex:daemon:leader", []byte(""), []byte("daemon-01"), 5000)
		log.Printf("CAS success: %v", success)
		if err != nil {
			log.Printf("CAS error: %v", err)
		}

		if success {
			log.Println("Leadership acquired")

			runCtx, cancel := context.WithCancel(ctx)

			go renewLock(runCtx, db)

			RunWatchStream(runCtx, db)

			cancel()
		}

		time.Sleep(2 * time.Second)
	}
}

func RunWatchStream(ctx context.Context, db KindClient) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			events, err := db.Watch(ctx, "vortex:metrics:")
			if err != nil {
				log.Printf("Failed to open watch stream: %v", err)
				return
			}

			for {
				event, err := events.Recv()
				if err != nil {
					log.Printf("Stream error or closed: %v", err)
					break
				}

				if event.OperationType == "PUT" {
					var m Metrics
					if err := json.Unmarshal(event.NewValue, &m); err != nil {
						log.Printf("Failed to parse metrics: %v", err)
						continue
					}

					log.Printf("METRIC EVENT client=%s rps=%d", m.ClientID, m.CurrentRPS)

					targetReplicas := m.CurrentRPS / 100
					if targetReplicas == 0 && m.CurrentRPS > 0 {
						targetReplicas = 1
					}

					RequestsTotal.WithLabelValues("200").Add(float64(m.CurrentRPS))

					log.Printf("Target replicas = %d/100 = %d", m.CurrentRPS, targetReplicas)

					active, exists := ClientCache[m.ClientID]
					if !exists || active != targetReplicas {
						ClientCache[m.ClientID] = targetReplicas

						totalNodes := 0
						for _, replicas := range ClientCache {
							totalNodes += replicas
						}
						ActiveNodesGauge.Set(float64(totalNodes))

						job := ScaleJob{
							ClientID: m.ClientID,
							Target:   targetReplicas,
						}
						jobData, _ := json.Marshal(job)

						queueKey := fmt.Sprintf("vortex:queue:scale:%s", m.ClientID)
						_, err := db.Put(ctx, queueKey, jobData)
						if err != nil {
							log.Printf("Failed to push ScaleJob: %v", err)
						} else {
							log.Printf("ScaleJob pushed to %s target=%d", queueKey, targetReplicas)
						}
					}
				}
			}
		}
	}
}

func renewLock(ctx context.Context, db KindClient) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			success, err := db.CAS(context.Background(), "vortex:daemon:leader", []byte("daemon-01"), []byte("daemon-01"), 5000)
			if err != nil || !success {
				log.Println("Failed to renew leadership lock!")
				return
			}
		}
	}
}

func NewKindClient(addr string) (KindClient, error) {
	return nil, fmt.Errorf("Kind client not implemented - use mock in tests")
}
