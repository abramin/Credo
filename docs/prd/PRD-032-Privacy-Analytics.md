# PRD-032: Privacy-Preserving Analytics

**Status:** Draft
**Priority:** P2 (Strategic)
**Owner:** Engineering Team
**Dependencies:** PRD-006 (Audit & Compliance), PRD-010 (Zero-Knowledge Proofs)

---

## 1. Overview

### Problem Statement

Businesses need analytics about their users to make informed decisions, but accessing individual user data creates privacy and compliance risks:

- **Privacy risk:** Raw data access exposes PII
- **Compliance risk:** GDPR requires data minimization
- **Trust risk:** Users don't want their data analyzed without consent
- **Security risk:** Centralized data access is an attack target

### Goals

- Enable **aggregate analytics without individual PII access**
- Implement **differential privacy** for query results
- Provide **query audit trail** showing who asked what
- Give users **transparency** into what queries touched their data
- Support **privacy budget** limiting total information disclosure

### Non-Goals

- Real-time analytics or streaming queries
- Machine learning model training
- Individual user behavior tracking
- Visualization or dashboarding UI

---

## 2. User Stories

**As a** business analyst
**I want to** get aggregate statistics about users
**So that** I can make data-driven decisions

**As a** compliance officer
**I want to** see all analytics queries that were run
**So that** I can audit data access patterns

**As a** user
**I want to** know what aggregate queries included my data
**So that** I understand how my data is used

**As a** privacy engineer
**I want to** enforce differential privacy on all queries
**So that** individual users can't be identified from results

---

## 3. Functional Requirements

### FR-1: Run Privacy-Preserving Query

**Endpoint:** `POST /analytics/query`

**Description:** Execute an aggregate query with differential privacy noise.

**Input:**
```json
{
  "query": {
    "type": "aggregate",
    "metrics": ["count", "avg_age", "verification_rate"],
    "filters": {
      "created_after": "2025-01-01",
      "verification_status": "verified"
    },
    "group_by": ["country"]
  },
  "privacy": {
    "epsilon": 1.0,
    "mechanism": "laplace"
  }
}
```

**Output (Success - 200):**
```json
{
  "query_id": "q_abc123",
  "results": [
    {
      "country": "US",
      "count": 1247,
      "avg_age": 34.2,
      "verification_rate": 0.78,
      "noise_applied": true
    },
    {
      "country": "UK",
      "count": 892,
      "avg_age": 31.8,
      "verification_rate": 0.82,
      "noise_applied": true
    }
  ],
  "metadata": {
    "epsilon_used": 1.0,
    "privacy_budget_remaining": 4.0,
    "min_group_size": 100,
    "suppressed_groups": 3
  }
}
```

### FR-2: Available Query Types

**Supported Aggregations:**
| Function | Description | Privacy Notes |
|----------|-------------|---------------|
| `count` | Count of records | Laplace noise added |
| `sum` | Sum of numeric field | Bounded sensitivity |
| `avg` | Average of numeric field | Derived from sum/count |
| `min`/`max` | Range statistics | Suppressed if group too small |
| `percentile` | Distribution percentiles | Requires larger epsilon |
| `distinct_count` | Unique value count | Uses HyperLogLog + noise |

### FR-3: Query Audit Trail

**Endpoint:** `GET /admin/analytics/audit`

**Output (Success - 200):**
```json
{
  "queries": [
    {
      "query_id": "q_abc123",
      "queried_by": "analyst@example.com",
      "query_type": "aggregate",
      "metrics": ["count", "avg_age"],
      "filters_applied": true,
      "epsilon_used": 1.0,
      "rows_touched": 15234,
      "timestamp": "2025-12-17T10:00:00Z"
    }
  ],
  "total_queries": 47,
  "total_epsilon_used": 23.5
}
```

### FR-4: User Transparency

**Endpoint:** `GET /me/analytics/usage`

**Description:** User sees which queries included their data.

**Output (Success - 200):**
```json
{
  "queries_involving_you": [
    {
      "query_id": "q_abc123",
      "query_date": "2025-12-17",
      "query_type": "aggregate",
      "purpose": "user_demographics",
      "your_data_fields": ["age_range", "country", "verification_status"],
      "result_type": "aggregate_only"
    }
  ],
  "total_queries": 12,
  "period": "last_90_days"
}
```

### FR-5: Privacy Budget Management

**Endpoint:** `GET /admin/analytics/budget`

**Output (Success - 200):**
```json
{
  "tenant_id": "tenant_123",
  "budget": {
    "total_epsilon": 10.0,
    "used_epsilon": 6.5,
    "remaining_epsilon": 3.5,
    "reset_at": "2026-01-01T00:00:00Z"
  },
  "usage_by_analyst": [
    {"analyst": "alice@example.com", "epsilon_used": 4.0},
    {"analyst": "bob@example.com", "epsilon_used": 2.5}
  ]
}
```

---

## 4. Technical Requirements

### TR-1: Differential Privacy Implementation

**Laplace Mechanism:**
```
noisy_result = true_result + Laplace(0, sensitivity/epsilon)
```

**Parameters:**
- **Epsilon (ε):** Privacy loss parameter (lower = more privacy)
- **Sensitivity:** Maximum change from adding/removing one record
- **Minimum group size:** Suppress results for groups < threshold

### TR-2: Query Restrictions

- Only aggregate queries allowed (no `SELECT *`)
- Filters cannot be too specific (min 100 matching records)
- Group-by results suppressed if group < 50 records
- No queries returning individual identifiers

### TR-3: Data Model

```
analytics_queries
├── query_id (PK)
├── tenant_id
├── queried_by
├── query_definition (JSONB)
├── epsilon_used
├── rows_touched
├── result_hash
├── timestamp
└── status

analytics_budget
├── tenant_id (PK)
├── total_epsilon
├── used_epsilon
├── reset_period
└── last_reset_at

user_query_touchpoints
├── user_id
├── query_id
├── fields_included []
├── timestamp
└── (no result data stored)
```

### TR-4: Query DSL

Restricted query language:
```yaml
type: aggregate
select:
  - function: count
    alias: user_count
  - function: avg
    field: age
    alias: avg_age
from: users
where:
  - field: created_at
    op: gte
    value: "2025-01-01"
group_by:
  - country
having:
  - field: user_count
    op: gte
    value: 100
```

### TR-5: SQL Query Patterns & Database Design

**Objective:** Demonstrate intermediate-to-advanced SQL capabilities for privacy-preserving analytics.

**Query Patterns Required:**

- **Aggregate Functions with Differential Privacy:**
  ```sql
  SELECT
    country,
    COUNT(*) + (random() * 2 - 1) * :noise_scale AS noisy_count,
    AVG(age) + (random() * 2 - 1) * :noise_scale AS noisy_avg_age,
    SUM(CASE WHEN verified = true THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0) AS verification_rate
  FROM users
  WHERE created_at >= :start_date
    AND tenant_id = :tenant_id
  GROUP BY country
  HAVING COUNT(*) >= :min_group_size;  -- Suppress small groups
  ```

- **Window Functions for Privacy Budget Tracking:**
  ```sql
  SELECT analyst_id, query_id, epsilon_used, timestamp,
         SUM(epsilon_used) OVER (
           PARTITION BY analyst_id
           ORDER BY timestamp
           ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
         ) AS cumulative_epsilon,
         :total_budget - SUM(epsilon_used) OVER (
           PARTITION BY analyst_id
           ORDER BY timestamp
         ) AS remaining_budget
  FROM analytics_queries
  WHERE tenant_id = :tenant_id
    AND timestamp >= :budget_reset_date;
  ```

- **CTE for User Touchpoint Tracking:**
  ```sql
  WITH query_fields AS (
    SELECT q.query_id, q.timestamp,
           jsonb_array_elements_text(q.query_definition->'metrics') AS metric,
           jsonb_array_elements_text(q.query_definition->'filters'->>'fields') AS filter_field
    FROM analytics_queries q
    WHERE q.timestamp > NOW() - INTERVAL '90 days'
  ),
  user_touchpoints AS (
    SELECT :user_id AS user_id, qf.query_id, qf.timestamp,
           array_agg(DISTINCT qf.metric) AS metrics_used,
           array_agg(DISTINCT qf.filter_field) AS filters_used
    FROM query_fields qf
    GROUP BY qf.query_id, qf.timestamp
  )
  SELECT * FROM user_touchpoints ORDER BY timestamp DESC;
  ```

- **Star Schema for Analytics (OLAP Pattern):**
  ```sql
  -- Dimension tables
  CREATE TABLE dim_users (
    user_key SERIAL PRIMARY KEY,
    user_id UUID UNIQUE,
    country TEXT,
    age_band TEXT,  -- '18-24', '25-34', etc. (no exact age)
    verification_status TEXT,
    created_date DATE
  );

  CREATE TABLE dim_time (
    time_key SERIAL PRIMARY KEY,
    full_date DATE UNIQUE,
    year INT, quarter INT, month INT, week INT, day_of_week INT
  );

  -- Fact table (aggregated, no PII)
  CREATE TABLE fact_analytics (
    id SERIAL PRIMARY KEY,
    time_key INT REFERENCES dim_time(time_key),
    country TEXT,
    age_band TEXT,
    user_count INT,
    verified_count INT,
    consent_count INT
  );

  -- OLAP query using star schema
  SELECT dt.year, dt.quarter, du.country, du.age_band,
         SUM(fa.user_count) AS total_users,
         SUM(fa.verified_count)::float / NULLIF(SUM(fa.user_count), 0) AS verification_rate
  FROM fact_analytics fa
  JOIN dim_time dt ON fa.time_key = dt.time_key
  JOIN dim_users du ON fa.country = du.country AND fa.age_band = du.age_band
  GROUP BY ROLLUP(dt.year, dt.quarter, du.country, du.age_band);
  ```

- **Materialized View for Pre-Aggregated Summaries:**
  ```sql
  CREATE MATERIALIZED VIEW privacy_safe_demographics AS
  SELECT
    DATE_TRUNC('month', created_at) AS cohort_month,
    country,
    CASE
      WHEN age < 25 THEN '18-24'
      WHEN age < 35 THEN '25-34'
      WHEN age < 45 THEN '35-44'
      ELSE '45+'
    END AS age_band,
    COUNT(*) AS user_count,
    COUNT(*) FILTER (WHERE verified = true) AS verified_count
  FROM users
  GROUP BY cohort_month, country, age_band
  HAVING COUNT(*) >= 50  -- k-anonymity threshold
  WITH DATA;

  CREATE INDEX ON privacy_safe_demographics (cohort_month, country);
  REFRESH MATERIALIZED VIEW CONCURRENTLY privacy_safe_demographics;
  ```

- **DISTINCT and HyperLogLog for Cardinality:**
  ```sql
  -- Exact distinct count (expensive)
  SELECT country, COUNT(DISTINCT user_id) AS exact_unique_users
  FROM consent_records
  GROUP BY country;

  -- Approximate using HyperLogLog (privacy-friendly, faster)
  SELECT country,
         hll_cardinality(hll_add_agg(hll_hash_text(user_id::text))) AS approx_unique_users
  FROM consent_records
  GROUP BY country;
  ```

**Database Design:**

- **Star Schema:** Dimension tables (`dim_users`, `dim_time`, `dim_location`) with fact table (`fact_analytics`)
- **No PII in Fact Tables:** Only aggregated counts, rates, and bucketed demographics
- **K-Anonymity Enforcement:** `HAVING COUNT(*) >= :k` on all GROUP BY queries
- **Partitioning:** Fact tables partitioned by month for efficient time-range queries
- **Materialized Views:** Pre-aggregate demographics with scheduled refresh (hourly/daily)

**Acceptance Criteria (SQL):**
- [ ] Aggregate queries add Laplace noise for differential privacy
- [ ] Privacy budget tracking uses window functions (cumulative sum)
- [ ] User touchpoint queries use CTEs with JSONB extraction
- [ ] Star schema separates dimensions from facts (OLAP pattern)
- [ ] Materialized views enforce k-anonymity (min group size)
- [ ] HyperLogLog used for approximate distinct counts
- [ ] All queries suppress groups below k-anonymity threshold

---

## 5. Acceptance Criteria

- [ ] Aggregate queries return differentially private results
- [ ] Small groups suppressed (< 50 records)
- [ ] Privacy budget tracked per tenant
- [ ] Queries blocked when budget exhausted
- [ ] All queries logged in audit trail
- [ ] Users can see which queries touched their data
- [ ] No individual records ever returned
- [ ] Epsilon configurable per query

---

## 6. Dependencies & Risks

### Dependencies
- PRD-006 (Audit) - Query audit logging
- PRD-010 (ZKP) - Optional ZKP for advanced proofs

### Risks
- **Utility vs Privacy tradeoff:** High privacy = low accuracy
  - *Mitigation:* Configurable epsilon, minimum sample sizes
- **Budget gaming:** Analysts running many low-epsilon queries
  - *Mitigation:* Per-analyst tracking, minimum epsilon per query

---

## Revision History

| Version | Date       | Author      | Changes                                                                                      |
| ------- | ---------- | ----------- | -------------------------------------------------------------------------------------------- |
| 1.1     | 2025-12-21 | Engineering | Added TR-5: SQL Query Patterns (aggregates, window functions, CTEs, star schema, OLAP, HLL) |
| 1.0     | 2025-12-17 | Engineering | Initial draft                                                                                |
