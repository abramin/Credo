package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	UsersCreated    prometheus.Counter
	ActiveSessions  prometheus.Gauge
	TokenRequests   prometheus.Counter
	AuthFailures    prometheus.Counter
	EndpointLatency *prometheus.HistogramVec
}

// New creates and registers all Prometheus metrics
func New() *Metrics {
	return &Metrics{
		UsersCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "id_gateway_users_created_total",
			Help: "Total number of users created in the system",
		}),
		ActiveSessions: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "id_gateway_active_sessions",
			Help: "Current number of active sessions",
		}),
		// 		- Token requests per minute (rate)
		TokenRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: "id_gateway_token_requests_total",
			Help: "Total number of token requests",
		}),
		// - Auth failures per minute (rate)
		AuthFailures: promauto.NewCounter(prometheus.CounterOpts{
			Name: "id_gateway_auth_failures_total",
			Help: "Total number of authentication failures",
		}),
		// - Latency per endpoint (histogram)
		EndpointLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "id_gateway_endpoint_latency_seconds",
			Help:    "Latency of endpoints in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"endpoint"}),
	}
}

// IncrementUsersCreated increments the users created counter by 1
func (m *Metrics) IncrementUsersCreated() {
	m.UsersCreated.Inc()
}

func (m *Metrics) IncrementActiveSessions(count int) {
	m.ActiveSessions.Add(float64(count))
}
func (m *Metrics) DecrementActiveSessions(count int) {
	m.ActiveSessions.Sub(float64(count))
}
func (m *Metrics) IncrementTokenRequests() {
	m.TokenRequests.Inc()
}
func (m *Metrics) IncrementAuthFailures() {
	m.AuthFailures.Inc()
}

// ObserveEndpointLatency records the latency for a given endpoint

func (m *Metrics) ObserveEndpointLatency(endpoint string, durationSeconds float64) {
	m.EndpointLatency.WithLabelValues(endpoint).Observe(durationSeconds)
}
