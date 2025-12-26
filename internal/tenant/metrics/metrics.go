package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics provides observability for the tenant module.
// Tracks tenant/client creation counts and critical path durations.
type Metrics struct {
	TenantCreated         prometheus.Counter
	ResolveClientDuration prometheus.Histogram
	CreateClientDuration  prometheus.Histogram
	GetTenantDuration     prometheus.Histogram
}

// New creates a new Metrics instance with all tenant module metrics registered.
func New() *Metrics {
	return &Metrics{
		TenantCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "credo_tenants_created_total",
			Help: "Total number of tenants created",
		}),
		ResolveClientDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "credo_resolve_client_duration_seconds",
			Help:    "Duration of ResolveClient operations (OAuth critical path)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}),
		CreateClientDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "credo_create_client_duration_seconds",
			Help:    "Duration of CreateClient operations (client registration path)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}),
		GetTenantDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "credo_get_tenant_duration_seconds",
			Help:    "Duration of GetTenant operations (tenant details with counts)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}),
	}
}

// IncrementTenantCreated records a successful tenant creation.
func (m *Metrics) IncrementTenantCreated() {
	m.TenantCreated.Inc()
}

// ObserveResolveClient records the duration of a ResolveClient operation.
// Call with time.Now() at the start of the operation.
func (m *Metrics) ObserveResolveClient(start time.Time) {
	m.ResolveClientDuration.Observe(time.Since(start).Seconds())
}

// ObserveCreateClient records the duration of a CreateClient operation.
// Call with time.Now() at the start of the operation.
func (m *Metrics) ObserveCreateClient(start time.Time) {
	m.CreateClientDuration.Observe(time.Since(start).Seconds())
}

// ObserveGetTenant records the duration of a GetTenant operation.
// Call with time.Now() at the start of the operation.
func (m *Metrics) ObserveGetTenant(start time.Time) {
	m.GetTenantDuration.Observe(time.Since(start).Seconds())
}
