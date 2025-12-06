# PRD-005B: ML-Based Risk Scoring

**Status:** Not Started
**Priority:** P2 (Medium - ML Showcase)
**Owner:** Engineering Team
**Dependencies:** PRD-005 (Decision Engine) complete
**Last Updated:** 2025-12-06

---

## 1. Overview

### Problem Statement
The current decision engine (PRD-005) uses simple rule-based logic. Real-world identity systems need risk assessment based on patterns that are difficult to encode as explicit rules. Machine learning can identify suspicious behavior patterns, fraud indicators, and anomalies that rule-based systems miss.

### Goals
- Add ML-based risk scoring to the decision engine
- Train a simple risk model on synthetic fraud data
- Combine rule-based + ML-based decision making
- Score users on risk level (0-100 scale)
- Provide explainable risk factors
- Demonstrate polyglot skills (Go + Python/ML)

### Non-Goals
- Production-grade fraud detection (this is a learning project)
- Real-time model training
- Complex deep learning models
- A/B testing different models
- Model drift detection
- Adversarial robustness

---

## 2. User Stories

**As a** decision engine
**I want to** assess risk based on learned patterns
**So that** I can catch fraud that rules miss

**As a** compliance officer
**I want to** understand why a user was flagged as high-risk
**So that** I can review and approve/deny manually

**As a** developer
**I want to** see ML integration in an identity system
**So that** I can learn how to productionize ML models

**As an** interviewer
**I want to** see that the candidate can work with ML/AI
**So that** I know they have relevant modern skills

---

## 3. Risk Factors (Features)

### User-Level Features
- **Account age:** Days since registration
- **Activity frequency:** Logins per week
- **Geographic consistency:** IP country matches claimed country
- **Device consistency:** Same device fingerprint over time

### Transaction-Level Features (from context)
- **National ID validation:** Citizen record valid/invalid
- **Sanctions status:** Listed on sanctions registry
- **Age verification:** User age matches claimed
- **Credential existence:** Has issued VCs

### Behavioral Features
- **Login velocity:** Logins per hour (rapid = suspicious)
- **Failed auth attempts:** Multiple failures before success
- **Time-of-day:** Unusual hours (3am access)
- **Request pattern:** Sequence of actions (normal vs suspicious)

---

## 4. Functional Requirements

### FR-1: Generate Risk Score
**Internal Function** (called by decision engine)

**Input:** `RiskInput`
```go
type RiskInput struct {
    UserID          string
    AccountAge      int     // days
    LoginFrequency  float64 // logins per week
    IPCountry       string
    DeviceID        string
    CitizenValid    bool
    SanctionsListed bool
    HasCredentials  bool
    RequestContext  map[string]any
}
```

**Output:** `RiskScore`
```go
type RiskScore struct {
    Score         float64            // 0-100 (0=safe, 100=high risk)
    Level         string             // "low", "medium", "high", "critical"
    Confidence    float64            // Model confidence 0-1
    Factors       []RiskFactor       // Explainability
    ModelVersion  string             // Which model generated this
    ComputedAt    time.Time
}

type RiskFactor struct {
    Feature     string  // "account_age", "sanctions_listed"
    Contribution float64 // How much this feature increased risk
    Value       any     // Actual feature value
}
```

**Business Logic:**
1. Extract features from RiskInput
2. Call ML service (Python) via HTTP/gRPC
3. Receive risk score (0-100)
4. Translate score to risk level
5. Extract feature contributions for explainability
6. Return RiskScore

---

### FR-2: Integrate Risk Score into Decision
**Enhancement to:** `POST /decision/evaluate` (PRD-005)

**Updated Response:**
```json
{
  "status": "pass_with_conditions",
  "reason": "medium_risk_detected",
  "conditions": ["manual_review"],
  "evidence": {
    "citizen_valid": true,
    "sanctions_listed": false,
    "has_credential": true
  },
  "risk_score": {
    "score": 62,
    "level": "medium",
    "confidence": 0.87,
    "factors": [
      {
        "feature": "account_age",
        "contribution": 15,
        "value": 2
      },
      {
        "feature": "login_velocity",
        "contribution": 25,
        "value": 12
      }
    ]
  },
  "evaluated_at": "2025-12-06T10:00:00Z"
}
```

**Updated Decision Rules:**
```
1. IF rule-based decision == FAIL → FAIL (rules take precedence)
2. IF risk_score >= 80 → FAIL (reason: "high_risk")
3. IF risk_score >= 60 → PASS_WITH_CONDITIONS (conditions: ["manual_review"])
4. IF risk_score < 60 → Continue with original decision logic
```

---

### FR-3: Train ML Model (Offline)
**Script:** `scripts/train_risk_model.py`

**Description:** Train a simple logistic regression or random forest model on synthetic fraud data.

**Training Data Format:**
```csv
account_age,login_frequency,citizen_valid,sanctions_listed,has_credentials,is_fraud
2,12.5,true,false,false,1
145,2.3,true,false,true,0
1,50.0,false,false,false,1
```

**Model Training Steps:**
1. Generate synthetic fraud dataset (10K samples)
2. Split train/test (80/20)
3. Train model (logistic regression or random forest)
4. Evaluate on test set (AUC, precision, recall)
5. Save model to disk (pickle or joblib)
6. Version model files

**Model Files:**
- `models/risk_model_v1.pkl` - Trained model
- `models/scaler_v1.pkl` - Feature scaler
- `models/metadata_v1.json` - Model info (features, version, metrics)

---

### FR-4: ML Service (Python)
**Service:** `services/ml-risk/app.py`

**Description:** Lightweight Python service that loads model and serves predictions.

**Endpoint:** `POST /predict`

**Input:**
```json
{
  "features": {
    "account_age": 2,
    "login_frequency": 12.5,
    "citizen_valid": 1,
    "sanctions_listed": 0,
    "has_credentials": 0
  }
}
```

**Output:**
```json
{
  "risk_score": 75.3,
  "confidence": 0.82,
  "feature_contributions": {
    "account_age": 18.5,
    "login_frequency": 32.1,
    "citizen_valid": -5.2,
    "sanctions_listed": 0.0,
    "has_credentials": -8.7
  },
  "model_version": "v1"
}
```

**Tech Stack:**
- Flask or FastAPI
- scikit-learn for model
- numpy/pandas for data
- Docker container for deployment

---

## 5. Technical Requirements

### TR-1: Risk Scoring Service (Go)

**Location:** `internal/ml/risk_service.go` (new package)

```go
type RiskService struct {
    mlClient *http.Client
    mlURL    string
}

func (s *RiskService) ScoreRisk(ctx context.Context, input RiskInput) (*RiskScore, error) {
    // 1. Extract features
    features := extractFeatures(input)
    
    // 2. Call ML service
    resp, err := s.callMLService(ctx, features)
    if err != nil {
        return nil, err
    }
    
    // 3. Convert to RiskScore
    return &RiskScore{
        Score:        resp.RiskScore,
        Level:        determineLevel(resp.RiskScore),
        Confidence:   resp.Confidence,
        Factors:      convertFactors(resp.FeatureContributions),
        ModelVersion: resp.ModelVersion,
        ComputedAt:   time.Now(),
    }, nil
}

func extractFeatures(input RiskInput) map[string]float64 {
    return map[string]float64{
        "account_age":       float64(input.AccountAge),
        "login_frequency":   input.LoginFrequency,
        "citizen_valid":     boolToFloat(input.CitizenValid),
        "sanctions_listed":  boolToFloat(input.SanctionsListed),
        "has_credentials":   boolToFloat(input.HasCredentials),
    }
}

func determineLevel(score float64) string {
    switch {
    case score >= 80:
        return "critical"
    case score >= 60:
        return "high"
    case score >= 40:
        return "medium"
    default:
        return "low"
    }
}
```

### TR-2: Decision Engine Integration

**Update:** `internal/decision/service.go`

```go
type Service struct {
    registry    *registry.Service
    vcStore     vc.Store
    auditor     audit.Publisher
    riskService *ml.RiskService  // NEW
    now         func() time.Time
}

func (s *Service) Evaluate(ctx context.Context, in DecisionInput) (DecisionOutcome, error) {
    // EXISTING rule-based checks
    if in.SanctionsListed {
        return DecisionOutcome{Status: DecisionFail, Reason: "sanctioned"}, nil
    }
    
    // NEW: ML risk scoring
    riskInput := ml.RiskInput{
        UserID:          in.UserID,
        AccountAge:      calculateAccountAge(in.UserID),
        LoginFrequency:  calculateLoginFrequency(in.UserID),
        CitizenValid:    in.CitizenValid,
        SanctionsListed: in.SanctionsListed,
        HasCredentials:  in.HasCredential,
    }
    
    riskScore, err := s.riskService.ScoreRisk(ctx, riskInput)
    if err != nil {
        // Log error but don't fail decision (graceful degradation)
        log.Warn("ML risk scoring failed", "error", err)
    }
    
    // NEW: Risk-based decision logic
    if riskScore != nil {
        if riskScore.Score >= 80 {
            return DecisionOutcome{
                Status:    DecisionFail,
                Reason:    "high_risk",
                RiskScore: riskScore,
            }, nil
        }
        if riskScore.Score >= 60 {
            return DecisionOutcome{
                Status:     DecisionPassWithConditions,
                Reason:     "medium_risk_detected",
                Conditions: []string{"manual_review"},
                RiskScore:  riskScore,
            }, nil
        }
    }
    
    // Continue with EXISTING rule-based logic
    if !in.CitizenValid {
        return DecisionOutcome{Status: DecisionFail, Reason: "invalid_citizen"}, nil
    }
    // ... rest of rules
}
```

### TR-3: ML Service (Python)

**Location:** `services/ml-risk/app.py` (new directory)

```python
from flask import Flask, request, jsonify
import joblib
import numpy as np

app = Flask(__name__)

# Load model on startup
model = joblib.load('models/risk_model_v1.pkl')
scaler = joblib.load('models/scaler_v1.pkl')

FEATURE_NAMES = ['account_age', 'login_frequency', 'citizen_valid', 
                 'sanctions_listed', 'has_credentials']

@app.route('/predict', methods=['POST'])
def predict():
    data = request.json
    features = data['features']
    
    # Extract features in correct order
    X = np.array([[
        features['account_age'],
        features['login_frequency'],
        features['citizen_valid'],
        features['sanctions_listed'],
        features['has_credentials']
    ]])
    
    # Scale features
    X_scaled = scaler.transform(X)
    
    # Predict probability of fraud
    risk_score = model.predict_proba(X_scaled)[0][1] * 100
    
    # Get feature contributions (for linear models)
    if hasattr(model, 'coef_'):
        contributions = {}
        for i, name in enumerate(FEATURE_NAMES):
            contributions[name] = float(model.coef_[0][i] * X_scaled[0][i])
    else:
        contributions = {}  # Random Forest doesn't have simple coefficients
    
    return jsonify({
        'risk_score': float(risk_score),
        'confidence': 0.8,  # Simplified for MVP
        'feature_contributions': contributions,
        'model_version': 'v1'
    })

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
```

**Dockerfile:**
```dockerfile
FROM python:3.10-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .
CMD ["python", "app.py"]
```

**requirements.txt:**
```
flask==3.0.0
scikit-learn==1.3.0
joblib==1.3.0
numpy==1.24.0
```

### TR-4: Data Generation Script

**Location:** `scripts/generate_fraud_data.py`

```python
import pandas as pd
import numpy as np

np.random.seed(42)

def generate_synthetic_data(n_samples=10000):
    data = []
    
    for i in range(n_samples):
        # Legitimate users (70%)
        if i < n_samples * 0.7:
            account_age = np.random.randint(30, 365)
            login_frequency = np.random.uniform(0.5, 5.0)
            citizen_valid = 1
            sanctions_listed = 0
            has_credentials = np.random.choice([0, 1], p=[0.3, 0.7])
            is_fraud = 0
        
        # Fraudsters (30%)
        else:
            account_age = np.random.randint(0, 10)
            login_frequency = np.random.uniform(10.0, 50.0)
            citizen_valid = np.random.choice([0, 1], p=[0.6, 0.4])
            sanctions_listed = np.random.choice([0, 1], p=[0.9, 0.1])
            has_credentials = 0
            is_fraud = 1
        
        data.append({
            'account_age': account_age,
            'login_frequency': login_frequency,
            'citizen_valid': citizen_valid,
            'sanctions_listed': sanctions_listed,
            'has_credentials': has_credentials,
            'is_fraud': is_fraud
        })
    
    df = pd.DataFrame(data)
    df.to_csv('data/fraud_data.csv', index=False)
    print(f"Generated {n_samples} samples")
    return df

if __name__ == '__main__':
    generate_synthetic_data()
```

### TR-5: Model Training Script

**Location:** `scripts/train_risk_model.py`

```python
import pandas as pd
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import StandardScaler
from sklearn.linear_model import LogisticRegression
from sklearn.metrics import classification_report, roc_auc_score
import joblib
import json

# Load data
df = pd.read_csv('data/fraud_data.csv')

# Split features and target
X = df.drop('is_fraud', axis=1)
y = df['is_fraud']

# Train/test split
X_train, X_test, y_train, y_test = train_test_split(
    X, y, test_size=0.2, random_state=42, stratify=y
)

# Scale features
scaler = StandardScaler()
X_train_scaled = scaler.fit_transform(X_train)
X_test_scaled = scaler.transform(X_test)

# Train model
model = LogisticRegression(random_state=42, max_iter=1000)
model.fit(X_train_scaled, y_train)

# Evaluate
y_pred = model.predict(X_test_scaled)
y_prob = model.predict_proba(X_test_scaled)[:, 1]

print("Classification Report:")
print(classification_report(y_test, y_pred))

auc = roc_auc_score(y_test, y_prob)
print(f"\nAUC-ROC: {auc:.3f}")

# Save model
joblib.dump(model, 'models/risk_model_v1.pkl')
joblib.dump(scaler, 'models/scaler_v1.pkl')

# Save metadata
metadata = {
    'model_version': 'v1',
    'model_type': 'LogisticRegression',
    'features': list(X.columns),
    'auc': float(auc),
    'trained_on': pd.Timestamp.now().isoformat()
}

with open('models/metadata_v1.json', 'w') as f:
    json.dump(metadata, f, indent=2)

print("\nModel saved successfully!")
```

---

## 6. Implementation Steps

### Phase 1: Data Generation & Model Training (4-6 hours)
1. Create `scripts/generate_fraud_data.py` and generate 10K samples
2. Create `scripts/train_risk_model.py` and train model
3. Evaluate model performance (aim for AUC > 0.85)
4. Save model files

### Phase 2: Python ML Service (3-4 hours)
1. Create `services/ml-risk/` directory
2. Implement Flask app with /predict endpoint
3. Dockerize ML service
4. Test locally (curl predictions)

### Phase 3: Go Risk Service (2-3 hours)
1. Create `internal/ml/` package
2. Implement RiskService with HTTP client
3. Unit tests for feature extraction
4. Integration test calling ML service

### Phase 4: Decision Engine Integration (2-3 hours)
1. Update DecisionInput to include risk features
2. Call RiskService in decision evaluation
3. Apply risk-based decision rules
4. Update response to include risk_score

### Phase 5: Docker Compose (1-2 hours)
1. Add ML service to docker-compose.yml
2. Configure service discovery
3. Test end-to-end flow

### Phase 6: Documentation (2-3 hours)
1. Document ML model approach
2. Explain feature engineering
3. Show how to retrain model
4. Performance benchmarks

---

## 7. Acceptance Criteria

- [ ] Synthetic fraud dataset generated (10K+ samples)
- [ ] ML model trained with AUC > 0.80
- [ ] Python ML service returns predictions
- [ ] Go service successfully calls ML service
- [ ] Decision engine incorporates risk scores
- [ ] High-risk users flagged appropriately
- [ ] Risk score includes explainability (feature contributions)
- [ ] ML service runs in Docker container
- [ ] End-to-end flow works with docker-compose
- [ ] Documentation explains ML approach
- [ ] Model files versioned and tracked

---

## 8. Testing

```bash
# Generate training data
python scripts/generate_fraud_data.py

# Train model
python scripts/train_risk_model.py
# Expected: AUC > 0.80, model saved

# Start ML service
cd services/ml-risk
docker build -t ml-risk .
docker run -p 5000:5000 ml-risk

# Test ML service directly
curl -X POST http://localhost:5000/predict \
  -H "Content-Type: application/json" \
  -d '{
    "features": {
      "account_age": 2,
      "login_frequency": 15.0,
      "citizen_valid": 0,
      "sanctions_listed": 0,
      "has_credentials": 0
    }
  }'
# Expected: {"risk_score": 78.5, "confidence": 0.8, ...}

# Test via decision engine
curl -X POST http://localhost:8080/decision/evaluate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "purpose": "age_verification",
    "context": {"national_id": "123456789"}
  }'
# Expected: Decision includes risk_score object

# Test high-risk user (should fail or require review)
# Create new account, trigger high-risk indicators
# Verify decision reflects risk level
```

---

## 9. Performance Considerations

- **ML service latency:** <100ms for prediction
- **Graceful degradation:** If ML service fails, fall back to rule-based only
- **Caching:** Cache risk scores for 5 minutes per user
- **Model size:** Keep model < 10MB for fast loading
- **Batch predictions:** Support batch scoring (future)

---

## 10. Model Monitoring (Future)

- Track prediction latency
- Monitor model confidence distribution
- Detect data drift (feature distributions change)
- A/B test model versions
- Collect feedback on false positives/negatives
- Retrain periodically with new data

---

## 11. Future Enhancements

- **Better models:** Random Forest, XGBoost, Neural Networks
- **More features:** Device fingerprinting, geolocation, behavioral sequences
- **Online learning:** Update model with real feedback
- **Model versioning:** A/B test multiple models
- **Explainability:** SHAP values for better interpretability
- **Real-time training:** Retrain on streaming data
- **Adversarial robustness:** Detect attempts to game the system

---

## 12. Resume Talking Points

This feature demonstrates:
- **Polyglot skills:** Go + Python integration
- **ML fundamentals:** Feature engineering, model training, evaluation
- **Production ML:** Model serving, versioning, monitoring
- **System design:** Microservices, graceful degradation
- **Explainability:** Risk factors for transparency

**Interview Answers:**
- "How would you scale this?" → Batch predictions, model caching, horizontal ML service scaling
- "How do you prevent adversarial attacks?" → Rate limiting, anomaly detection on feature distributions
- "How do you improve the model?" → Collect labeled data, retrain periodically, A/B test new models

---

## 13. References

- [scikit-learn Documentation](https://scikit-learn.org/stable/)
- [Flask Quickstart](https://flask.palletsprojects.com/en/3.0.x/quickstart/)
- [Model Serving Best Practices](https://www.tensorflow.org/tfx/guide/serving)
- Existing Code: PRD-005 (Decision Engine)

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-12-06 | Engineering Team | Initial PRD |
