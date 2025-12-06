# PRD-007B: ML-Based Risk Scoring

**Status:** Not Started
**Priority:** P2 (Medium)
**Owner:** Engineering Team
**Last Updated:** 2025-12-06
**Dependencies:** PRD-005 (Decision Engine), PRD-006 (Audit & Compliance)

---

## 1. Overview

### Problem Statement
Current decision engine uses static rule-based logic. While powerful, it cannot learn from patterns, adapt to emerging threats, or provide probabilistic risk assessments. Manual rule maintenance becomes burdensome as fraud patterns evolve.

### Goals
- Build ML-based risk scoring system that learns from historical decisions
- Provide risk scores (0.0-1.0) for identity verification requests
- Train on historical audit data (approvals, rejections, fraud reports)
- Support multiple risk dimensions (identity fraud, account takeover, synthetic identity)
- Enable hybrid approach: ML scores feed into existing rules engine
- Provide explainability: which features contributed to risk score
- Support model retraining as new data arrives

### Non-Goals
- Real-time model training (batch training sufficient)
- Deep learning models (gradient boosting/random forests sufficient)
- Automatic model deployment (manual review required)
- Integration with external fraud databases
- Behavioral biometrics
- Graph-based fraud detection
- Federated learning across tenants

---

## 2. User Stories

### As a System Administrator
- I want to enable ML risk scoring for my tenant
- I want to configure risk thresholds that trigger manual review
- I want to see model performance metrics (precision, recall, AUC)
- I want to retrain models on new data periodically
- I want to understand which features contribute to risk scores
- I want to disable ML scoring if accuracy degrades

### As a Compliance Officer
- I want to audit risk score decisions
- I want to ensure risk models don't introduce bias
- I want to see model explainability for rejected applications
- I want to compare ML decisions against rule-based decisions

### As a Developer (Integration)
- I want risk scores returned alongside decision results
- I want to query historical risk scores for analysis
- I want to report fraud/false positives to improve model
- I want to understand confidence intervals on risk scores

---

## 3. Technical Design

### 3.1 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Decision Engine                        │
│  ┌──────────────────┐         ┌─────────────────────────┐  │
│  │  Rules Engine    │────────▶│  Risk Aggregator        │  │
│  │  (existing)      │         │  (combines scores)      │  │
│  └──────────────────┘         └─────────────────────────┘  │
│           │                              │                  │
│           ▼                              ▼                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │            ML Risk Scoring Service                    │  │
│  │  ┌────────────┐  ┌────────────┐  ┌───────────────┐  │  │
│  │  │ Feature    │  │ Model      │  │ Explainability│  │  │
│  │  │ Extractor  │─▶│ Inference  │─▶│ Generator     │  │  │
│  │  └────────────┘  └────────────┘  └───────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
              ┌──────────────────────────────┐
              │   Model Training Pipeline    │
              │  ┌────────────────────────┐  │
              │  │ Historical Data        │  │
              │  │ Extraction             │  │
              │  └────────┬───────────────┘  │
              │           ▼                  │
              │  ┌────────────────────────┐  │
              │  │ Feature Engineering    │  │
              │  └────────┬───────────────┘  │
              │           ▼                  │
              │  ┌────────────────────────┐  │
              │  │ Model Training         │  │
              │  │ (Gradient Boosting)    │  │
              │  └────────┬───────────────┘  │
              │           ▼                  │
              │  ┌────────────────────────┐  │
              │  │ Model Evaluation       │  │
              │  └────────┬───────────────┘  │
              │           ▼                  │
              │  ┌────────────────────────┐  │
              │  │ Model Serialization    │  │
              │  └────────────────────────┘  │
              └──────────────────────────────┘
```

### 3.2 Data Model

**Risk Score**
```go
type RiskScore struct {
    ID               string              `json:"id"`
    TenantID         string              `json:"tenant_id"`
    RequestID        string              `json:"request_id"`
    OverallScore     float64             `json:"overall_score"`      // 0.0-1.0
    RiskDimensions   map[string]float64  `json:"risk_dimensions"`    // fraud, synthetic, etc.
    ModelVersion     string              `json:"model_version"`
    Features         map[string]float64  `json:"features"`           // feature values
    FeatureImportance map[string]float64 `json:"feature_importance"` // SHAP values
    Confidence       float64             `json:"confidence"`         // 0.0-1.0
    Timestamp        time.Time           `json:"timestamp"`
}
```

**ML Model Metadata**
```go
type MLModel struct {
    ID              string            `json:"id"`
    TenantID        string            `json:"tenant_id"`
    ModelType       string            `json:"model_type"`        // "gradient_boost", "random_forest"
    Version         string            `json:"version"`
    TrainedAt       time.Time         `json:"trained_at"`
    TrainingDataset DatasetInfo       `json:"training_dataset"`
    Metrics         ModelMetrics      `json:"metrics"`
    IsActive        bool              `json:"is_active"`
    Configuration   map[string]string `json:"configuration"`
}

type ModelMetrics struct {
    Precision    float64 `json:"precision"`
    Recall       float64 `json:"recall"`
    F1Score      float64 `json:"f1_score"`
    AUC          float64 `json:"auc"`
    Accuracy     float64 `json:"accuracy"`
    SampleSize   int     `json:"sample_size"`
    ConfusionMatrix [][]int `json:"confusion_matrix"`
}
```

**Feature Definition**
```go
type FeatureDefinition struct {
    Name        string   `json:"name"`
    Type        string   `json:"type"`           // "numeric", "categorical", "boolean"
    Source      string   `json:"source"`         // "credential", "audit", "derived"
    Transform   string   `json:"transform"`      // "log", "standardize", "one_hot"
    Description string   `json:"description"`
}
```

### 3.3 Features for Risk Scoring

**Identity Features**
- Email domain age (days since domain registration)
- Email provider type (free vs. corporate)
- Phone number country vs. claimed country match
- Document type and issuing country
- Time since identity document issuance
- Credential confidence scores

**Behavioral Features**
- Time of day (hour of request)
- Day of week
- Device fingerprint (if available)
- IP address geolocation vs. claimed location
- Consent grant patterns (speed, comprehensiveness)

**Historical Features**
- Previous verification attempts for same identity
- Previous fraud reports for similar patterns
- Tenant-specific fraud rate
- Time since last verification request

**Derived Features**
- Number of active consents
- Credential diversity (multiple sources)
- Identity document age
- Request velocity (requests per hour/day)

### 3.4 API Design

#### Score Request
```http
POST /api/v1/ml/score
Authorization: Bearer {token}
Content-Type: application/json

{
  "request_id": "req_abc123",
  "features": {
    "email_domain_age_days": 3650,
    "phone_country_match": true,
    "credential_confidence": 0.95,
    "time_of_day_hour": 14,
    "previous_verifications": 0
  },
  "options": {
    "include_explanation": true,
    "risk_dimensions": ["fraud", "synthetic_identity"]
  }
}
```

**Response:**
```json
{
  "risk_score": {
    "id": "score_xyz789",
    "request_id": "req_abc123",
    "overall_score": 0.23,
    "risk_dimensions": {
      "identity_fraud": 0.15,
      "synthetic_identity": 0.31,
      "account_takeover": 0.08
    },
    "model_version": "v1.2.0",
    "confidence": 0.87,
    "timestamp": "2025-12-06T10:30:00Z"
  },
  "explanation": {
    "top_features": [
      {
        "name": "email_domain_age_days",
        "value": 3650,
        "contribution": -0.12,
        "interpretation": "reduces risk"
      },
      {
        "name": "previous_verifications",
        "value": 0,
        "contribution": 0.08,
        "interpretation": "increases risk"
      },
      {
        "name": "credential_confidence",
        "value": 0.95,
        "contribution": -0.05,
        "interpretation": "reduces risk"
      }
    ]
  }
}
```

#### Train Model
```http
POST /api/v1/ml/models/train
Authorization: Bearer {admin_token}
Content-Type: application/json

{
  "tenant_id": "tenant_abc",
  "model_type": "gradient_boost",
  "training_window": {
    "start": "2025-01-01T00:00:00Z",
    "end": "2025-12-01T00:00:00Z"
  },
  "configuration": {
    "n_estimators": 100,
    "max_depth": 6,
    "learning_rate": 0.1
  }
}
```

**Response:**
```json
{
  "job_id": "job_train_123",
  "status": "running",
  "started_at": "2025-12-06T10:30:00Z",
  "estimated_duration_minutes": 15
}
```

#### Get Model Metrics
```http
GET /api/v1/ml/models/{model_id}/metrics
Authorization: Bearer {token}
```

**Response:**
```json
{
  "model_id": "model_xyz",
  "version": "v1.2.0",
  "metrics": {
    "precision": 0.92,
    "recall": 0.87,
    "f1_score": 0.89,
    "auc": 0.94,
    "accuracy": 0.91,
    "sample_size": 12543
  },
  "confusion_matrix": [
    [8234, 456],
    [387, 3466]
  ],
  "trained_at": "2025-11-15T08:00:00Z"
}
```

#### Report Fraud
```http
POST /api/v1/ml/feedback
Authorization: Bearer {token}
Content-Type: application/json

{
  "request_id": "req_abc123",
  "score_id": "score_xyz789",
  "feedback_type": "fraud_confirmed",
  "notes": "Identity theft confirmed by user"
}
```

### 3.5 ML Implementation Stack

**Language & Framework**
- Python 3.11+ for ML training pipeline
- Go service wraps Python model via gRPC or REST
- Alternative: ONNX Runtime for Go-native inference

**ML Libraries**
- **Primary:** XGBoost or LightGBM (gradient boosting)
- **Alternative:** scikit-learn RandomForestClassifier
- **Explainability:** SHAP (SHapley Additive exPlanations)
- **Feature Engineering:** pandas, numpy

**Model Serving**
- Serialize models as pickle or ONNX format
- Load into Go service at startup
- In-memory inference (no external calls)
- Model versioning: load multiple models, route by tenant config

**Training Pipeline**
- Batch training (daily/weekly schedule)
- Extract features from audit logs and credential DB
- Train/test split (80/20)
- Cross-validation for hyperparameter tuning
- Store models in versioned storage (S3 or local filesystem)

### 3.6 Integration with Rules Engine

**Hybrid Approach**
```yaml
rules:
  - name: "High Risk Requires Manual Review"
    condition: "risk_score > 0.7"
    action: "require_manual_review"
    
  - name: "Medium Risk with Low Confidence"
    condition: "risk_score > 0.4 AND risk_confidence < 0.6"
    action: "require_additional_verification"
    
  - name: "Low Risk Auto-Approve"
    condition: "risk_score < 0.2 AND all_consents_granted"
    action: "approve"
```

**Decision Flow**
1. Extract features from request
2. Call ML scoring service
3. Inject risk score into rules engine context
4. Rules engine evaluates with risk_score available
5. Log risk score in audit trail
6. Return decision + risk score to caller

---

## 4. Implementation Plan

### Phase 1: Foundation (Week 1-2)
- [ ] Design feature extraction pipeline
- [ ] Create training data export from audit logs
- [ ] Build Python training script
- [ ] Train initial model on synthetic/sample data
- [ ] Define RiskScore data model
- [ ] Create risk_scores table in PostgreSQL

### Phase 2: Model Serving (Week 3)
- [ ] Implement model loading in Go service
- [ ] Create /ml/score API endpoint
- [ ] Integrate feature extraction from request context
- [ ] Add model version tracking
- [ ] Implement basic explainability (feature contributions)

### Phase 3: Integration (Week 4)
- [ ] Integrate ML scoring into decision engine
- [ ] Update rules DSL to support risk_score variable
- [ ] Add risk score logging to audit trail
- [ ] Create admin API for model management
- [ ] Implement /ml/feedback endpoint

### Phase 4: Training Pipeline (Week 5-6)
- [ ] Automate feature extraction from historical data
- [ ] Implement model training job
- [ ] Add model evaluation and validation
- [ ] Create model versioning and rollback
- [ ] Build monitoring dashboard for model performance

### Phase 5: Production Readiness (Week 7)
- [ ] Add model performance degradation detection
- [ ] Implement A/B testing framework (rule-based vs. ML)
- [ ] Create documentation for compliance officers
- [ ] Add bias detection and fairness metrics
- [ ] Performance testing and optimization

---

## 5. Testing Strategy

### Unit Tests
- Feature extraction correctness
- Model serialization/deserialization
- SHAP value calculation
- API request/response validation

### Integration Tests
- End-to-end scoring flow
- Model training pipeline
- Feedback loop
- Rules engine integration

### Model Testing
- Performance on holdout dataset
- Adversarial examples (edge cases)
- Bias testing across demographics (if available)
- Model drift detection

### Load Tests
- Inference latency (target: <50ms p99)
- Concurrent scoring requests
- Model loading time
- Memory usage with multiple models

---

## 6. Success Metrics

### Model Performance
- **Precision:** >0.85 (fraud detection)
- **Recall:** >0.80 (catch most fraud)
- **AUC-ROC:** >0.90
- **Inference Latency:** <50ms p99

### Business Metrics
- Reduction in false positives (fewer manual reviews)
- Fraud detection rate improvement
- Time to detection (catch fraud faster)
- Manual review workload reduction

### Operational Metrics
- Model staleness (days since last training)
- Feature coverage (% of requests with all features)
- Explainability clarity (human comprehensibility)

---

## 7. Security & Privacy Considerations

### Data Privacy
- Minimize PII in features (use derived/anonymized features)
- Secure model storage (encrypted at rest)
- Access control for training data
- Audit log all model training and scoring events

### Model Security
- Validate model integrity (checksum verification)
- Prevent model poisoning (validate training data)
- Rate limiting on scoring endpoint
- Sandbox Python execution if dynamic

### Bias & Fairness
- Regular bias audits
- Ensure features don't proxy for protected attributes
- Document feature selection rationale
- Enable tenant-specific model tuning

---

## 8. Documentation Requirements

### For Developers
- Feature engineering guide
- Model training runbook
- API integration examples
- Troubleshooting guide

### For Administrators
- Model configuration guide
- Performance monitoring dashboard
- Retraining procedures
- Model rollback procedures

### For Compliance
- Model explainability documentation
- Bias testing results
- Data retention policies
- Audit trail specifications

---

## 9. Open Questions

1. **Labeling Strategy:** How do we get ground truth labels (fraud/not fraud)?
   - Rely on manual review outcomes?
   - User-reported fraud?
   - External fraud database integration?

2. **Model Update Frequency:** How often to retrain?
   - Daily, weekly, monthly?
   - Triggered by performance degradation?

3. **Multi-Tenancy:** Shared model vs. tenant-specific models?
   - Start with shared, allow opt-in for tenant-specific?

4. **Feature Store:** Do we need a dedicated feature store?
   - Or is feature extraction at request time sufficient?

5. **Cold Start:** How to score requests with no historical data?
   - Fallback to rule-based only?
   - Use industry benchmarks?

---

## 10. Future Enhancements

- **Graph-based fraud detection** (detect fraud rings)
- **Behavioral biometrics** (typing patterns, mouse movements)
- **Federated learning** (train across tenants without sharing data)
- **Deep learning models** (if data volume justifies)
- **Real-time model updates** (online learning)
- **External fraud database integration** (cross-reference known bad actors)
- **Anomaly detection** (unsupervised learning for novel fraud patterns)

---

## 11. References

- XGBoost Documentation: https://xgboost.readthedocs.io/
- SHAP for Explainability: https://github.com/slundberg/shap
- ONNX Runtime: https://onnxruntime.ai/
- scikit-learn: https://scikit-learn.org/
- Fairness in ML: https://fairmlbook.org/
