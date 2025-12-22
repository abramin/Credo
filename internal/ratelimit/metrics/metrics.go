package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	RateLimitAuthFailures          prometheus.Counter
	RateLimitAuthLockoutsTotal     prometheus.Counter
	RateLimitAuthLockedIdentifiers prometheus.Gauge
}

func New() *Metrics {
	return &Metrics{
		RateLimitAuthFailures: promauto.NewCounter(prometheus.CounterOpts{
			Name: "credo_ratelimit_auth_failures_recorded_total",
			Help: "Total number of auth failures recorded for rate limiting",
		}),
		RateLimitAuthLockoutsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "credo_ratelimit_auth_lockouts_total",
			Help:        "Total number of auth lockouts recorded for rate limiting",
			ConstLabels: prometheus.Labels{"type": "lockoutType"},
		}),
		RateLimitAuthLockedIdentifiers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "credo_ratelimit_auth_locked_identifiers",
			Help: "Current number of hard locked identifiers due to rate limiting",
		}),
	}
}

func (m *Metrics) IncrementAuthFailures() {
	m.RateLimitAuthFailures.Inc()
}

func (m *Metrics) IncrementAuthLockouts() {
	m.RateLimitAuthLockoutsTotal.Inc()
}

func (m *Metrics) SetLockedIdentifiers(count int) {
	m.RateLimitAuthLockedIdentifiers.Set(float64(count))
}
