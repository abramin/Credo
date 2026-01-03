package ops

import (
	"math/rand"
	"sync"
)

// Sampler provides configurable sampling for ops events.
// High-volume events can be sampled down to reduce storage and processing costs.
type Sampler struct {
	mu           sync.RWMutex
	defaultRate  float64
	rateByAction map[string]float64
}

// NewSampler creates a sampler with the given default rate.
// Rate should be between 0.0 (sample nothing) and 1.0 (sample everything).
func NewSampler(defaultRate float64) *Sampler {
	if defaultRate < 0 {
		defaultRate = 0
	}
	if defaultRate > 1 {
		defaultRate = 1
	}
	return &Sampler{
		defaultRate:  defaultRate,
		rateByAction: make(map[string]float64),
	}
}

// ShouldSample returns true if the event should be sampled (kept).
func (s *Sampler) ShouldSample(action string) bool {
	rate := s.rateFor(action)
	return rand.Float64() < rate //nolint:gosec // sampling doesn't need crypto rand
}

// SetRate sets the sample rate for a specific action.
// Use this to override the default for high-volume actions.
func (s *Sampler) SetRate(action string, rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rateByAction[action] = rate
}

// SetDefaultRate changes the default sample rate.
func (s *Sampler) SetDefaultRate(rate float64) {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultRate = rate
}

func (s *Sampler) rateFor(action string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rate, ok := s.rateByAction[action]; ok {
		return rate
	}
	return s.defaultRate
}
