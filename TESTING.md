# 🧪 Testing Vortex

Welcome to the official testing documentation for **Vortex**. As a dynamic, auto-scaling reverse proxy and cluster manager, testing Vortex involves more than just ensuring the code compiles—it requires proving that it can gracefully handle massive traffic spikes, dynamically scale nodes, and resiliently recover from failures.

This guide provides a comprehensive walkthrough for unit testing the codebase, validating the auto-scaler under heavy load, triggering the rate limiter, executing chaos engineering tests, and monitoring the system via Grafana.

---

## Prerequisites

Before running the tests, ensure you have the following tools installed:

- **Go (1.20+)**: Required for running standard unit tests.
- **Docker & Docker Compose**: Required for integration testing and running observability tools (Prometheus & Grafana).
- **hey** (or **k6**): A dedicated load testing tool for simulating concurrent traffic.

**Quick install for `hey` (by rakyll):**
```bash
go install github.com/rakyll/hey@latest
```
*(Ensure `$(go env GOPATH)/bin` is in your `$PATH`)*

---

## 🧪 Unit Testing

We use Go's built-in testing framework for standard unit tests across all packages (e.g., `daemon`, `loadbalancer`, etc.).

To run the full suite of unit tests with coverage, execute the following command at the root of the repository:

```bash
# Run all tests, enable race detector, and output coverage report
go test ./... -v -race -cover
```

**Expected Output:**
You should see output indicating tests passing for each package, along with a coverage percentage.
```text
ok  	github.com/vortex/daemon      0.042s  coverage: 85.0% of statements
ok  	github.com/vortex/loadbalancer 0.031s  coverage: 92.3% of statements
PASS
```

---

## 🔥 Load Testing & Auto-Scaling Validation

*This is the most critical test for Vortex.* The goal is to simulate a massive traffic spike and watch the Vortex engine automatically spawn new backend nodes to handle the load.

**1. Start the Vortex cluster:**
```bash
# Start Vortex (listening on port 8000 by default)
go run main.go
```

**2. Blast the load balancer with concurrent traffic:**
Using `hey`, send 10,000 requests to the load balancer with 200 concurrent workers:

```bash
# -n: Total number of requests (10000)
# -c: Number of concurrent workers (200)
hey -n 10000 -c 200 http://localhost:8000/
```

**Expected Observation:**
Watch the primary terminal running `main.go`. As the connection threshold is breached, the daemon will recognize the traffic spike and instantly spawn new nodes.

*Logs you should see:*
```log
[INFO] Traffic spike detected. Connection count: 1542
[INFO] Scaling up cluster...
[SUCCESS] Spawned new backend node on port 3001
[SUCCESS] Spawned new backend node on port 3002
[INFO] Rebalancing traffic using Round-Robin...
```
In the `hey` output, expect a `Status code distribution` of predominantly `200 OK`.

---

## 🛡️ Rate Limiting Tests

Vortex incorporates strict rate limiting to protect the cluster from abuse (e.g., DDoS attacks). To test this, you must exceed the maximum requests-per-second threshold configured in `vortex.yaml`.

**1. Run a rapid-fire curl loop:**
```bash
# Fire 50 consecutive requests as fast as possible
for i in {1..50}; do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8000/; done
```

**Expected Observation:**
The first few requests will return `200` (OK). Once the rate limit threshold is hit, you will see `429` (Too Many Requests) HTTP status codes blocking further traffic.

*Sample Output:*
```text
200
200
200
429
429
429
```

---

## 💥 Chaos Testing (Health Checks)

Vortex continuously monitors the health of backend nodes. We need to verify that if a node catastrophically fails, Vortex immediately sidelines it without dropping ongoing user traffic.

**1. Setup the scenario:**
Start Vortex and allow it to spawn a few nodes (e.g., ports `3001`, `3002`, `3003`).

**2. Induce a failure:**
Manually kill one of the backend node processes (or stop its Docker container).

```bash
# Find the PID of the node running on port 3002
lsof -i :3002

# Kill the process
kill -9 <PID>
```

**3. Watch Vortex heal the cluster:**
Instantly send a few regular curl requests: `curl http://localhost:8000/`.

**Expected Observation:**
Vortex's health checker will ping the dead node, fail, and drop it from the Round-Robin pool. Traffic will flawlessly route to the surviving nodes (`3001` and `3003`).

*Logs you should see:*
```log
[WARN] Health check failed for node: localhost:3002
[ERROR] Node localhost:3002 is DEAD. Removing from active routing pool.
[INFO] Traffic successfully routed to healthy node: localhost:3001
```

---

## 📊 Observability Validation

Vortex is instrumented with Prometheus and Grafana for real-time traffic and state observability.

1. Ensure the Docker observability stack is running:
   ```bash
   docker-compose up -d prometheus grafana
   ```
2. Open Grafana in your browser: [http://localhost:3000](http://localhost:3000)
3. Navigate to the **Vortex Overview** dashboard.

While executing the **Load Testing** or **Chaos Testing** commands above, observe the Grafana dashboard. You will visually confirm:
- **Traffic Throughput:** Huge spikes in RPS (Requests Per Second).
- **Active Cluster Size:** The node count dynamically climbing as `hey` blasts the server, and decreasing when the traffic stops.
- **Error Rates:** Spikes corresponding to the `429` errors triggered during the Rate Limiting test, or momentary blips during the Chaos Test.

---

> **Note:** Testing is a continuous discipline. Whenever you add new middleware, load-balancing algorithms, or core engine tuning, ensure you write the corresponding unit tests and document new integration paths here!
