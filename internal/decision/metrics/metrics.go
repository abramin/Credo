package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics provides observability for the decision module.
type Metrics struct {
	// Evidence gathering latencies by source
	EvidenceLatency *prometheus.HistogramVec

	// Decision outcomes by status and purpose
	DecisionOutcome *prometheus.CounterVec

	// Overall evaluation latency
	EvaluateLatency prometheus.Histogram
}

// New creates a new Metrics instance with all decision module metrics registered.
func New() *Metrics {
	return &Metrics{
		EvidenceLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "credo_decision_evidence_duration_seconds",
			Help:    "Duration of evidence gathering operations by source",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
		}, []string{"source"}), // source: "citizen", "sanctions", "credential"

		DecisionOutcome: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "credo_decision_outcomes_total",
			Help: "Total decision outcomes by status and purpose",
		}, []string{"status", "purpose"}),

		EvaluateLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "credo_decision_evaluate_duration_seconds",
			Help:    "Duration of full decision evaluation including evidence gathering",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		}),
	}
}

// ObserveEvidenceLatency records the duration of fetching evidence from a source.
func (m *Metrics) ObserveEvidenceLatency(source string, d time.Duration) {
	if m != nil {
		m.EvidenceLatency.WithLabelValues(source).Observe(d.Seconds())
	}
}

// IncrementOutcome records a decision outcome.
func (m *Metrics) IncrementOutcome(status, purpose string) {
	if m != nil {
		m.DecisionOutcome.WithLabelValues(status, purpose).Inc()
	}
}

// ObserveEvaluateLatency records the total evaluation duration.
func (m *Metrics) ObserveEvaluateLatency(d time.Duration) {
	if m != nil {
		m.EvaluateLatency.Observe(d.Seconds())
	}
}
