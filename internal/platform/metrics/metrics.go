package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	UsersCreated prometheus.Counter
}

// New creates and registers all Prometheus metrics
func New() *Metrics {
	return &Metrics{
		UsersCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "id_gateway_users_created_total",
			Help: "Total number of users created in the system",
		}),
	}
}

// IncrementUsersCreated increments the users created counter by 1
func (m *Metrics) IncrementUsersCreated() {
	m.UsersCreated.Inc()
}
