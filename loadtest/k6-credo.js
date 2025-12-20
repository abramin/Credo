// k6 Load Test Suite for Credo OAuth Server
// Run: k6 run loadtest/k6-credo.js
//
// Environment variables:
//   BASE_URL       - Server URL (default: http://localhost:8080)
//   CLIENT_ID      - OAuth client ID
//   CLIENT_SECRET  - OAuth client secret (if confidential client)
//   ADMIN_TOKEN    - Admin API token for setup
//   SCENARIO       - Which scenario to run: token_refresh | consent_burst | mixed_load | all

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// Custom metrics
const tokenRefreshLatency = new Trend('token_refresh_latency', true);
const consentGrantLatency = new Trend('consent_grant_latency', true);
const sessionListLatency = new Trend('session_list_latency', true);
const tokenErrors = new Counter('token_errors');
const consentErrors = new Counter('consent_errors');
const errorRate = new Rate('error_rate');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const CLIENT_ID = __ENV.CLIENT_ID || 'test-client';
const SCENARIO = __ENV.SCENARIO || 'all';

// Scenario configurations
export const options = {
  scenarios: {
    // Scenario 1: Token Refresh Storm
    // Tests mutex contention under concurrent token refresh load
    token_refresh_storm: {
      executor: 'constant-arrival-rate',
      rate: 100,                    // 100 requests per second
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 50,
      maxVUs: 200,
      exec: 'tokenRefreshScenario',
      startTime: '0s',
      tags: { scenario: 'token_refresh' },
    },

    // Scenario 2: Consent Grant Burst
    // Tests consent service throughput with multi-purpose grants
    consent_burst: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      preAllocatedVUs: 20,
      maxVUs: 100,
      stages: [
        { duration: '1m', target: 50 },   // Ramp up
        { duration: '3m', target: 50 },   // Sustained load
        { duration: '1m', target: 0 },    // Ramp down
      ],
      exec: 'consentBurstScenario',
      startTime: '0s',
      tags: { scenario: 'consent_burst' },
    },

    // Scenario 3: Mixed Load (Read + Write contention)
    // Tests read performance during write contention
    mixed_load: {
      executor: 'constant-vus',
      vus: 50,
      duration: '5m',
      exec: 'mixedLoadScenario',
      startTime: '0s',
      tags: { scenario: 'mixed_load' },
    },
  },

  thresholds: {
    // Token refresh: p95 < 200ms, error rate < 0.1%
    'token_refresh_latency{scenario:token_refresh}': ['p(95)<200'],
    'error_rate{scenario:token_refresh}': ['rate<0.001'],

    // Consent grants: p95 < 300ms
    'consent_grant_latency{scenario:consent_burst}': ['p(95)<300'],

    // Mixed load: both reads and writes should be responsive
    'session_list_latency{scenario:mixed_load}': ['p(95)<100'],
    'token_refresh_latency{scenario:mixed_load}': ['p(95)<300'],
  },
};

// Filter scenarios based on SCENARIO env var
if (SCENARIO !== 'all') {
  const selectedScenario = options.scenarios[SCENARIO];
  if (selectedScenario) {
    options.scenarios = { [SCENARIO]: selectedScenario };
  }
}

// Setup: Create test users and get tokens
export function setup() {
  console.log(`Starting load test against ${BASE_URL}`);
  console.log(`Running scenario: ${SCENARIO}`);

  // In a real test, you would:
  // 1. Create test users via admin API
  // 2. Perform authorization flow to get tokens
  // 3. Return tokens for use in scenarios

  // For now, return placeholder data
  // Replace with actual token acquisition logic
  return {
    tokens: generateTestTokens(100),
    users: generateTestUsers(100),
  };
}

// Generate placeholder tokens (replace with real token acquisition)
function generateTestTokens(count) {
  const tokens = [];
  for (let i = 0; i < count; i++) {
    tokens.push({
      accessToken: `test-access-token-${i}`,
      refreshToken: `test-refresh-token-${i}`,
      userId: `user-${i}`,
    });
  }
  return tokens;
}

function generateTestUsers(count) {
  const users = [];
  for (let i = 0; i < count; i++) {
    users.push({
      id: `user-${i}`,
      email: `user${i}@test.com`,
    });
  }
  return users;
}

// Scenario 1: Token Refresh Storm
// Purpose: Validate mutex contention under concurrent token refresh load
export function tokenRefreshScenario(data) {
  const tokenIndex = __VU % data.tokens.length;
  const token = data.tokens[tokenIndex];

  const payload = {
    grant_type: 'refresh_token',
    refresh_token: token.refreshToken,
    client_id: CLIENT_ID,
  };

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: { name: 'token_refresh' },
  };

  const startTime = Date.now();
  const res = http.post(
    `${BASE_URL}/auth/token`,
    JSON.stringify(payload),
    params
  );
  const duration = Date.now() - startTime;

  tokenRefreshLatency.add(duration);

  const success = check(res, {
    'token refresh status is 200': (r) => r.status === 200,
    'token refresh has access_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token !== undefined;
      } catch {
        return false;
      }
    },
  });

  if (!success) {
    tokenErrors.add(1);
    errorRate.add(1);
    console.log(`Token refresh failed: ${res.status} - ${res.body}`);
  } else {
    errorRate.add(0);
    // Update token for next iteration if successful
    try {
      const body = JSON.parse(res.body);
      if (body.refresh_token) {
        data.tokens[tokenIndex].refreshToken = body.refresh_token;
      }
    } catch {
      // Ignore parse errors
    }
  }
}

// Scenario 2: Consent Grant Burst
// Purpose: Validate consent service throughput with multi-purpose grants
export function consentBurstScenario(data) {
  const userIndex = __VU % data.users.length;
  const user = data.users[userIndex];

  // Grant multiple purposes in one request
  const purposes = ['marketing', 'analytics', 'personalization', 'third_party_sharing'];
  const selectedPurposes = purposes.slice(0, Math.floor(Math.random() * purposes.length) + 1);

  const payload = {
    purposes: selectedPurposes,
  };

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${data.tokens[userIndex].accessToken}`,
    },
    tags: { name: 'consent_grant' },
  };

  const startTime = Date.now();
  const res = http.post(
    `${BASE_URL}/consent/grant`,
    JSON.stringify(payload),
    params
  );
  const duration = Date.now() - startTime;

  consentGrantLatency.add(duration);

  const success = check(res, {
    'consent grant status is 200 or 201': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    consentErrors.add(1);
    errorRate.add(1);
    if (res.status !== 401) { // Ignore auth errors in test mode
      console.log(`Consent grant failed: ${res.status} - ${res.body}`);
    }
  } else {
    errorRate.add(0);
  }

  sleep(0.1); // Small delay between requests
}

// Scenario 3: Mixed Load
// Purpose: Validate read performance during write contention
export function mixedLoadScenario(data) {
  const userIndex = __VU % data.users.length;
  const token = data.tokens[userIndex];

  group('mixed_operations', () => {
    // 70% reads (session listing), 30% writes (token refresh)
    if (Math.random() < 0.7) {
      // Read operation: List sessions
      const params = {
        headers: {
          'Authorization': `Bearer ${token.accessToken}`,
        },
        tags: { name: 'session_list' },
      };

      const startTime = Date.now();
      const res = http.get(`${BASE_URL}/auth/sessions`, params);
      const duration = Date.now() - startTime;

      sessionListLatency.add(duration);

      check(res, {
        'session list status is 200': (r) => r.status === 200,
      });
    } else {
      // Write operation: Token refresh
      const payload = {
        grant_type: 'refresh_token',
        refresh_token: token.refreshToken,
        client_id: CLIENT_ID,
      };

      const params = {
        headers: {
          'Content-Type': 'application/json',
        },
        tags: { name: 'token_refresh' },
      };

      const startTime = Date.now();
      const res = http.post(
        `${BASE_URL}/auth/token`,
        JSON.stringify(payload),
        params
      );
      const duration = Date.now() - startTime;

      tokenRefreshLatency.add(duration);

      const success = check(res, {
        'token refresh status is 200': (r) => r.status === 200,
      });

      if (success) {
        try {
          const body = JSON.parse(res.body);
          if (body.refresh_token) {
            data.tokens[userIndex].refreshToken = body.refresh_token;
          }
        } catch {
          // Ignore
        }
      }
    }
  });

  sleep(0.05); // 50ms between operations
}

// Teardown: Cleanup test data
export function teardown(data) {
  console.log('Load test complete');
  console.log(`Total tokens tested: ${data.tokens.length}`);
}

// Default function (required by k6)
export default function (data) {
  // This runs when no specific scenario is selected
  tokenRefreshScenario(data);
}
