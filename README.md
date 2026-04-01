<div align="center">
  <img src="https://via.placeholder.com/150?text=Vortex" alt="Vortex Logo" width="150" height="150" />
  <h1>Vortex</h1>
  <p><b>A highly dynamic, auto-scaling reverse proxy and cluster manager built for resilience and speed.</b></p>
  
  <p>
    <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
    <img src="https://img.shields.io/badge/Gin-008ECF?style=for-the-badge&logo=gin&logoColor=white" alt="Gin" />
    <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker" />
    <img src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" alt="License: MIT" />
  </p>
</div>

---

## 🌌 Overview

**Vortex** is a robust, dynamic reverse proxy, rate limiter, and cluster manager built entirely in **Go**. Engineered to handle fluctuating traffic seamlessly, Vortex dynamically spawns or terminates backend nodes based on real-time traffic thresholds. By leveraging the **Gin** framework and **yaml.v3** for declarative configuration, Vortex provides an enterprise-grade load balancing solution that is highly configurable, inherently resilient, and fully observable out of the box.

---

## ✨ Key Features

- **⚡ Auto-Scaling**: Intelligently spawns or kills backend nodes dynamically governed by live traffic thresholds.
- **⚖️ Load Balancing**: Distributes incoming traffic efficiently using a robust **Round-Robin** algorithm.
- **🛡️ Rate Limiting**: Protects your services from abuse and traffic spikes with built-in, configurable limits.
- **❤️ Health Checks**: Ensures high availability by continuously monitoring backend node health via a background daemon.
- **⚙️ Declarative Config**: Easy and version-controllable configuration using a single `vortex.yaml` file.
- **📊 Observability**: Fully integrated with **Prometheus** and **Grafana** for deep insights and analytics.

---

## 🏗️ Architecture

The Vortex architecture routes incoming requests through a robust chain of limiters and balancers, while a background daemon continuously monitors health and scales the cluster up or down based on load.

```mermaid
flowchart TD
    %% Custom Colors and Styling
    classDef client fill:#f9f9f9,stroke:#333,stroke-width:2px;
    classDef proxy fill:#e1f5fe,stroke:#0288d1,stroke-width:2px;
    classDef backend fill:#e8f5e9,stroke:#388e3c,stroke-width:2px;
    classDef daemon fill:#fff3e0,stroke:#f57c00,stroke-width:2px;
    classDef monitor fill:#fce4ec,stroke:#c2185b,stroke-width:2px;

    Client([Client Requests]) -->|HTTP/HTTPS| RL{Rate Limiter}
    
    RL -->|Allowed| Eng(Load Balancer Engine)
    RL -->|Blocked| Block([429 Too Many Requests])
    
    subgraph Vortex Clusterস্থল
        Eng --> Node1[Backend Node 1]
        Eng --> Node2[Backend Node 2]
        Eng --> NodeN[Backend Node N]
    end

    subgraph Background Daemons
        SD[[Scaling Daemon]]
        HC[[Health Checker]]
    end

    SD -.->|Auto-Scale| Vortex Clusterস্থল
    HC -.->|Monitor/Evict| Vortex Clusterস্থল
    
    %% Observability
    Vortex Clusterস্থল -.-> Prom[(Prometheus)]
    Prom --> Graf[Grafana Dashboard]

    %% Apply Classes
    class Client client;
    class RL,Eng proxy;
    class Node1,Node2,NodeN backend;
    class SD,HC daemon;
    class Prom,Graf monitor;
```

---

## 📂 Project Structure

```text
vortex/
├── daemon/                  # Background auto-scaling and health checker logic
├── docker-compose.yml       # Production stack orchestration
├── grafana/                 # Grafana dashboards and provisioning configs
├── loadbalancer/            # Round-Robin routing and engine core
├── main.go                  # Application entry point
├── ratelimiter/             # Traffic parsing and rate limit enforcement
└── vortex.yaml              # Declarative cluster configuration
```

---

## 🚀 Getting Started

You can run Vortex either via a quick local installation or deployed as a fully dockerized production stack.

### 💻 Local Installation

Ensure you have **Go 1.20+** installed, then simply clone and run:

```bash
# Clone the repository
git clone https://github.com/yourusername/vortex.git
cd vortex

# Run the proxy locally
go run main.go
```

### 🐳 Production Stack

For a complete deployment with observability, deploy Vortex using **Docker Compose**:

```bash
# Build and run in detached mode
docker compose up -d --build
```

**Expected Ports:**
- **Load Balancer**: `8080`
- **Grafana**: `3000` (Default login: `admin`/`admin`)
- **Prometheus**: `9090`

---

## 🛠️ Configuration

Configure the proxy using the declarative `vortex.yaml` file. Set up cluster size boundaries, scaling triggers, and routing paths effortlessly.

```yaml
# vortex.yaml
cluster:
  max_nodes: 10
  min_nodes: 2
  strategy: round_robin

scaling_triggers:
  cpu_threshold_percent: 80
  request_rate_per_sec: 1000
  scale_up_step: 2
  scale_down_step: 1

rate_limit:
  enabled: true
  requests_per_ip: 100
  window_seconds: 60
```

---

## 🧪 Testing & Monitoring

Once your stack is running, you can test the load balancing by hitting the proxy:

```bash
# Send a test request to the load balancer
curl -i http://localhost:8080/
```

**Observability**:
Open your browser to `http://localhost:3000` to access the **Grafana Dashboard**. The built-in dashboard will visualize live analytics from **Prometheus**, showing the active cluster size, request throughput, and auto-scaling events as they happen.

---

## 🤝 Contribution

Contributions make the open-source community an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

<div align="center">
  Built by <b>Nikhil</b>  
</div>
