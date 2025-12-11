package orchestrator

// import (
// 	"context"
// 	"fmt"
// 	"sync"
// 	"time"

// 	"credo/internal/evidence/registry/providers"
// )

// // LookupStrategy defines how to select providers for a lookup
// type LookupStrategy string

// const (
// 	// StrategyPrimary uses only the primary provider
// 	StrategyPrimary LookupStrategy = "primary"

// 	// StrategyFallback tries primary, then falls back to secondary on failure
// 	StrategyFallback LookupStrategy = "fallback"

// 	// StrategyParallel queries all providers in parallel and merges results
// 	StrategyParallel LookupStrategy = "parallel"

// 	// StrategyVoting queries multiple providers and uses majority vote
// 	StrategyVoting LookupStrategy = "voting"
// )

// // CorrelationRule defines how to merge evidence from multiple sources
// type CorrelationRule interface {
// 	// Merge combines evidence from multiple providers
// 	Merge(evidence []*providers.Evidence) (*providers.Evidence, error)

// 	// Applicable checks if this rule applies to the given evidence types
// 	Applicable(types []providers.ProviderType) bool
// }

// // ProviderChain defines a sequence of providers with fallback logic
// type ProviderChain struct {
// 	Primary   string   // Primary provider ID
// 	Secondary []string // Fallback provider IDs
// 	Timeout   time.Duration
// }

// // OrchestratorConfig configures the evidence orchestrator
// type OrchestratorConfig struct {
// 	Registry        *providers.ProviderRegistry
// 	DefaultStrategy LookupStrategy
// 	DefaultTimeout  time.Duration

// 	// Chains defines provider preferences by evidence type
// 	Chains map[providers.ProviderType]ProviderChain

// 	// Rules defines how to correlate multi-source evidence
// 	Rules []CorrelationRule
// }

// // Orchestrator coordinates multi-source evidence gathering
// type Orchestrator struct {
// 	registry *providers.ProviderRegistry
// 	chains   map[providers.ProviderType]ProviderChain
// 	rules    []CorrelationRule
// 	strategy LookupStrategy
// 	timeout  time.Duration
// }

// // NewOrchestrator creates a new evidence orchestrator
// func NewOrchestrator(cfg OrchestratorConfig) *Orchestrator {
// 	return &Orchestrator{
// 		registry: cfg.Registry,
// 		chains:   cfg.Chains,
// 		rules:    cfg.Rules,
// 		strategy: applyDefaultStrategy(cfg.DefaultStrategy),
// 		timeout:  applyDefaultTimeout(cfg.DefaultTimeout),
// 	}
// }

// func applyDefaultStrategy(strategy LookupStrategy) LookupStrategy {
// 	if strategy == "" {
// 		return StrategyFallback
// 	}
// 	return strategy
// }

// func applyDefaultTimeout(timeout time.Duration) time.Duration {
// 	if timeout == 0 {
// 		return 5 * time.Second
// 	}
// 	return timeout
// }

// // LookupRequest describes what evidence to gather
// type LookupRequest struct {
// 	Types    []providers.ProviderType // What types of evidence to gather
// 	Filters  map[string]string        // Input filters (national_id, etc.)
// 	Strategy LookupStrategy           // Override default strategy
// 	Timeout  time.Duration            // Override default timeout
// }

// // LookupResult contains all gathered evidence
// type LookupResult struct {
// 	Evidence []*providers.Evidence
// 	Errors   map[string]error // Provider ID -> error
// }

// func newLookupResult() *LookupResult {
// 	return &LookupResult{
// 		Evidence: make([]*providers.Evidence, 0),
// 		Errors:   make(map[string]error),
// 	}
// }

// func (r *LookupResult) recordError(providerID string, err error) {
// 	r.Errors[providerID] = err
// }

// func (r *LookupResult) addEvidence(evidence *providers.Evidence) {
// 	r.Evidence = append(r.Evidence, evidence)
// }

// func (r *LookupResult) hasNoEvidence() bool {
// 	return len(r.Evidence) == 0
// }

// func (r *LookupResult) hasErrors() bool {
// 	return len(r.Errors) > 0
// }

// func (r *LookupResult) toError() error {
// 	if r.hasNoEvidence() && r.hasErrors() {
// 		return providers.ErrAllProvidersFailed
// 	}
// 	return nil
// }

// // Lookup gathers evidence according to the request
// func (o *Orchestrator) Lookup(ctx context.Context, req LookupRequest) (*LookupResult, error) {
// 	ctx = o.applyTimeout(ctx, req.Timeout)
// 	strategy := o.selectStrategy(req.Strategy)

// 	return o.executeStrategy(ctx, req, strategy)
// }

// func (o *Orchestrator) applyTimeout(ctx context.Context, requestTimeout time.Duration) context.Context {
// 	timeout := requestTimeout
// 	if timeout == 0 {
// 		timeout = o.timeout
// 	}

// 	ctx, _ = context.WithTimeout(ctx, timeout)
// 	return ctx
// }

// func (o *Orchestrator) selectStrategy(requestStrategy LookupStrategy) LookupStrategy {
// 	if requestStrategy == "" {
// 		return o.strategy
// 	}
// 	return requestStrategy
// }

// func (o *Orchestrator) executeStrategy(ctx context.Context, req LookupRequest, strategy LookupStrategy) (*LookupResult, error) {
// 	switch strategy {
// 	case StrategyPrimary:
// 		return o.lookupPrimary(ctx, req)
// 	case StrategyFallback:
// 		return o.lookupFallback(ctx, req)
// 	case StrategyParallel:
// 		return o.lookupParallel(ctx, req)
// 	case StrategyVoting:
// 		return o.lookupVoting(ctx, req)
// 	default:
// 		return nil, fmt.Errorf("unknown strategy: %s", strategy)
// 	}
// }

// // ============================================================================
// // STRATEGY: Primary - Uses only the primary provider for each type
// // ============================================================================

// func (o *Orchestrator) lookupPrimary(ctx context.Context, req LookupRequest) (*LookupResult, error) {
// 	result := newLookupResult()

// 	for _, providerType := range req.Types {
// 		o.lookupFromPrimaryProvider(ctx, providerType, req.Filters, result)
// 	}

// 	return result, result.toError()
// }

// func (o *Orchestrator) lookupFromPrimaryProvider(
// 	ctx context.Context,
// 	providerType providers.ProviderType,
// 	filters map[string]string,
// 	result *LookupResult,
// ) {
// 	primaryProviderID := o.findPrimaryProvider(providerType)
// 	if primaryProviderID == "" {
// 		result.recordError("no-provider", providers.ErrNoProvidersAvailable)
// 		return
// 	}

// 	evidence, err := o.queryProvider(ctx, primaryProviderID, filters)
// 	if err != nil {
// 		result.recordError(primaryProviderID, err)
// 		return
// 	}

// 	result.addEvidence(evidence)
// }

// func (o *Orchestrator) findPrimaryProvider(providerType providers.ProviderType) string {
// 	// Check if there's a configured chain for this provider type
// 	if chain, exists := o.chains[providerType]; exists {
// 		return chain.Primary
// 	}

// 	// No chain defined, find any provider of this type
// 	availableProviders := o.registry.ListByType(providerType)
// 	if len(availableProviders) == 0 {
// 		return ""
// 	}

// 	return availableProviders[0].ID()
// }

// func (o *Orchestrator) queryProvider(
// 	ctx context.Context,
// 	providerID string,
// 	filters map[string]string,
// ) (*providers.Evidence, error) {
// 	provider, exists := o.registry.Get(providerID)
// 	if !exists {
// 		return nil, providers.ErrProviderNotFound
// 	}

// 	return provider.Lookup(ctx, filters)
// }

// // ============================================================================
// // STRATEGY: Fallback - Tries primary, then secondary on failure
// // ============================================================================

// func (o *Orchestrator) lookupFallback(ctx context.Context, req LookupRequest) (*LookupResult, error) {
// 	result := newLookupResult()

// 	for _, providerType := range req.Types {
// 		o.lookupWithFallback(ctx, providerType, req.Filters, result)
// 	}

// 	return result, result.toError()
// }

// func (o *Orchestrator) lookupWithFallback(
// 	ctx context.Context,
// 	providerType providers.ProviderType,
// 	filters map[string]string,
// 	result *LookupResult,
// ) {
// 	chain := o.getOrCreateChain(providerType)
// 	if chain.Primary == "" {
// 		result.recordError("no-provider", providers.ErrNoProvidersAvailable)
// 		return
// 	}

// 	// Try primary provider first
// 	if o.tryPrimaryProvider(ctx, chain.Primary, filters, result) {
// 		return // Success
// 	}

// 	// Try fallback providers if primary failed
// 	o.tryFallbackProviders(ctx, chain.Secondary, filters, result)
// }

// func (o *Orchestrator) getOrCreateChain(providerType providers.ProviderType) ProviderChain {
// 	if chain, exists := o.chains[providerType]; exists {
// 		return chain
// 	}

// 	// Create a chain from available providers
// 	availableProviders := o.registry.ListByType(providerType)
// 	if len(availableProviders) == 0 {
// 		return ProviderChain{}
// 	}

// 	return ProviderChain{Primary: availableProviders[0].ID()}
// }

// func (o *Orchestrator) tryPrimaryProvider(
// 	ctx context.Context,
// 	primaryID string,
// 	filters map[string]string,
// 	result *LookupResult,
// ) bool {
// 	evidence, err := o.queryProvider(ctx, primaryID, filters)
// 	if err == nil {
// 		result.addEvidence(evidence)
// 		return true
// 	}

// 	result.recordError(primaryID, err)
// 	return false
// }

// func (o *Orchestrator) tryFallbackProviders(
// 	ctx context.Context,
// 	fallbackIDs []string,
// 	filters map[string]string,
// 	result *LookupResult,
// ) {
// 	for _, fallbackID := range fallbackIDs {
// 		evidence, err := o.queryProvider(ctx, fallbackID, filters)
// 		if err == nil {
// 			result.addEvidence(evidence)
// 			return // Success, stop trying
// 		}

// 		result.recordError(fallbackID, err)
// 	}
// }

// // ============================================================================
// // STRATEGY: Parallel - Queries all providers in parallel
// // ============================================================================

// func (o *Orchestrator) lookupParallel(ctx context.Context, req LookupRequest) (*LookupResult, error) {
// 	result := o.queryAllProvidersInParallel(ctx, req.Types, req.Filters)
// 	result = o.applyCorrelationRules(result, req.Types)

// 	return result, result.toError()
// }

// func (o *Orchestrator) queryAllProvidersInParallel(
// 	ctx context.Context,
// 	providerTypes []providers.ProviderType,
// 	filters map[string]string,
// ) *LookupResult {
// 	result := newLookupResult()
// 	collector := newThreadSafeCollector()

// 	providerList := o.findAllProvidersForTypes(providerTypes)
// 	collector.queryInParallel(ctx, providerList, filters)

// 	result.Evidence = collector.evidence
// 	result.Errors = collector.errors

// 	return result
// }

// func (o *Orchestrator) findAllProvidersForTypes(types []providers.ProviderType) []providers.Provider {
// 	allProviders := make([]providers.Provider, 0)

// 	for _, providerType := range types {
// 		providers := o.registry.ListByType(providerType)
// 		allProviders = append(allProviders, providers...)
// 	}

// 	return allProviders
// }

// // threadSafeCollector handles parallel evidence collection with proper synchronization
// type threadSafeCollector struct {
// 	evidence []*providers.Evidence
// 	errors   map[string]error
// 	mu       sync.Mutex
// 	wg       sync.WaitGroup
// }

// func newThreadSafeCollector() *threadSafeCollector {
// 	return &threadSafeCollector{
// 		evidence: make([]*providers.Evidence, 0),
// 		errors:   make(map[string]error),
// 	}
// }

// func (c *threadSafeCollector) queryInParallel(
// 	ctx context.Context,
// 	providers []providers.Provider,
// 	filters map[string]string,
// ) {
// 	for _, provider := range providers {
// 		c.wg.Add(1)
// 		go c.queryProviderAsync(ctx, provider, filters)
// 	}

// 	c.wg.Wait()
// }

// func (c *threadSafeCollector) queryProviderAsync(
// 	ctx context.Context,
// 	provider providers.Provider,
// 	filters map[string]string,
// ) {
// 	defer c.wg.Done()

// 	evidence, err := provider.Lookup(ctx, filters)

// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	if err != nil {
// 		c.errors[provider.ID()] = err
// 	} else {
// 		c.evidence = append(c.evidence, evidence)
// 	}
// }

// func (o *Orchestrator) applyCorrelationRules(result *LookupResult, requestedTypes []providers.ProviderType) *LookupResult {
// 	if !o.shouldApplyCorrelation(result) {
// 		return result
// 	}

// 	evidenceTypes := extractProviderTypes(result.Evidence)
// 	applicableRule := o.findApplicableRule(evidenceTypes)

// 	if applicableRule == nil {
// 		return result
// 	}

// 	return o.mergeEvidenceWithRule(result, applicableRule)
// }

// func (o *Orchestrator) shouldApplyCorrelation(result *LookupResult) bool {
// 	return len(o.rules) > 0 && len(result.Evidence) > 1
// }

// func extractProviderTypes(evidence []*providers.Evidence) []providers.ProviderType {
// 	types := make([]providers.ProviderType, 0, len(evidence))
// 	for _, e := range evidence {
// 		types = append(types, e.ProviderType)
// 	}
// 	return types
// }

// func (o *Orchestrator) findApplicableRule(types []providers.ProviderType) CorrelationRule {
// 	for _, rule := range o.rules {
// 		if rule.Applicable(types) {
// 			return rule
// 		}
// 	}
// 	return nil
// }

// func (o *Orchestrator) mergeEvidenceWithRule(result *LookupResult, rule CorrelationRule) *LookupResult {
// 	merged, err := rule.Merge(result.Evidence)
// 	if err != nil {
// 		return result // Keep original evidence if merge fails
// 	}

// 	result.Evidence = []*providers.Evidence{merged}
// 	return result
// }

// // ============================================================================
// // STRATEGY: Voting - Uses majority vote based on confidence scores
// // ============================================================================

// func (o *Orchestrator) lookupVoting(ctx context.Context, req LookupRequest) (*LookupResult, error) {
// 	// First collect evidence from all providers
// 	result := o.queryAllProvidersInParallel(ctx, req.Types, req.Filters)

// 	// Select highest confidence evidence per type
// 	result.Evidence = selectHighestConfidenceEvidence(result.Evidence)

// 	return result, result.toError()
// }

// func selectHighestConfidenceEvidence(allEvidence []*providers.Evidence) []*providers.Evidence {
// 	bestByType := findBestEvidencePerType(allEvidence)
// 	return convertMapToSlice(bestByType)
// }

// func findBestEvidencePerType(evidence []*providers.Evidence) map[providers.ProviderType]*providers.Evidence {
// 	bestByType := make(map[providers.ProviderType]*providers.Evidence)

// 	for _, currentEvidence := range evidence {
// 		providerType := currentEvidence.ProviderType
// 		existingBest := bestByType[providerType]

// 		if shouldReplaceExisting(existingBest, currentEvidence) {
// 			bestByType[providerType] = currentEvidence
// 		}
// 	}

// 	return bestByType
// }

// func shouldReplaceExisting(existing, candidate *providers.Evidence) bool {
// 	return existing == nil || candidate.Confidence > existing.Confidence
// }

// func convertMapToSlice(evidenceMap map[providers.ProviderType]*providers.Evidence) []*providers.Evidence {
// 	result := make([]*providers.Evidence, 0, len(evidenceMap))
// 	for _, evidence := range evidenceMap {
// 		result = append(result, evidence)
// 	}
// 	return result
// }

// // ============================================================================
// // Health Check - Verifies all registered providers are healthy
// // ============================================================================

// func (o *Orchestrator) HealthCheck(ctx context.Context) map[string]error {
// 	allProviders := o.registry.All()
// 	healthResults := make(map[string]error, len(allProviders))

// 	for _, provider := range allProviders {
// 		healthResults[provider.ID()] = provider.Health(ctx)
// 	}

// 	return healthResults
// }
//
