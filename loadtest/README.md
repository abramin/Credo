# Credo Load Testing

Load test suite using [k6](https://k6.io) to validate performance improvements.

## Installation

```bash
# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6

# Docker
docker pull grafana/k6
```

## Quick Smoke Test

Verify the server is responding before running full load tests:

```bash
# Start the server
make run

# In another terminal, run smoke test
k6 run loadtest/k6-quick.js
```

## Full Load Test Suite

### Prerequisites

1. Start the server:

```bash
make run
# or
docker compose up
```

2. Create test data by running the E2E tests (requires server running):

```bash
# Terminal 1: Start server
make run

# Terminal 2: Run E2E tests to create users/sessions
make e2e

# Verify test data exists
curl http://localhost:8081/admin/users -H "X-Admin-Token: $ADMIN_TOKEN"
```

### Run Scenarios

```bash
# Run all scenarios
k6 run loadtest/k6-credo.js

# Run specific scenario
SCENARIO=token_refresh_storm k6 run loadtest/k6-credo.js
SCENARIO=consent_burst k6 run loadtest/k6-credo.js
SCENARIO=mixed_load k6 run loadtest/k6-credo.js

# With custom configuration
BASE_URL=http://localhost:8080 \
CLIENT_ID=my-client \
k6 run loadtest/k6-credo.js
```

## Scenarios

### 1. Token Refresh Storm

**Purpose**: Validate mutex contention under concurrent token refresh load

- **Load**: 100 req/sec for 5 minutes
- **VUs**: 50-200 virtual users
- **Target**: p95 latency < 200ms, error rate < 0.1%

### 2. Consent Grant Burst

**Purpose**: Validate consent service throughput with multi-purpose grants

- **Load**: Ramp from 10 to 50 req/sec over 5 minutes
- **VUs**: 20-100 virtual users
- **Target**: p95 latency < 300ms

### 3. Mixed Load (Read/Write)

**Purpose**: Validate read performance during write contention

- **Load**: 50 concurrent users, 70% reads / 30% writes
- **Duration**: 5 minutes
- **Target**: List p95 < 100ms, Refresh p95 < 300ms

## Metrics Collected

| Metric                  | Description                     |
| ----------------------- | ------------------------------- |
| `token_refresh_latency` | Token refresh endpoint latency  |
| `consent_grant_latency` | Consent grant endpoint latency  |
| `session_list_latency`  | Session list endpoint latency   |
| `token_errors`          | Count of token refresh failures |
| `consent_errors`        | Count of consent grant failures |
| `error_rate`            | Overall error rate              |

## Viewing Results

### Terminal Output

k6 prints summary statistics after each run.

### JSON Output

```bash
k6 run --out json=results.json loadtest/k6-credo.js
```

### InfluxDB + Grafana

```bash
# Start InfluxDB
docker run -d --name influxdb -p 8086:8086 influxdb:1.8

# Run with InfluxDB output
k6 run --out influxdb=http://localhost:8086/k6 loadtest/k6-credo.js
```

## Interpreting Results

### Healthy Results

```
     token_refresh_latency...: avg=45ms   p(95)=120ms  p(99)=180ms
     error_rate...............: 0.02%
     http_reqs................: 30000   100/s
```

### Concerning Results

```
     token_refresh_latency...: avg=450ms  p(95)=1200ms p(99)=2500ms
     error_rate...............: 5.2%
     http_reqs................: 15000   50/s
```

**Signs of mutex contention**:

- p99 >> p95 (long tail latencies)
- Throughput plateaus despite adding VUs
- Error rate increases under load

## Profiling During Load Tests

While running load tests, capture profiling data:

```bash
# CPU profile (30 seconds)
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# Mutex contention
go tool pprof http://localhost:8080/debug/pprof/mutex

# Block profile (goroutine blocking)
go tool pprof http://localhost:8080/debug/pprof/block

# Goroutine count
curl http://localhost:8080/debug/pprof/goroutine?debug=1 | head -20
```

## Tips

1. **Warm up first**: Run a quick smoke test before full load
2. **Isolate scenarios**: Run one scenario at a time for cleaner metrics
3. **Monitor system resources**: Watch CPU, memory, goroutines during tests
4. **Repeat tests**: Run 3x and average results to reduce noise
5. **Test realistic data**: Use production-like token counts and user distributions
