// Shared OAuth2 Helper Functions
// Used across all demo pages to keep code DRY

/**
 * Determine API base URL based on environment
 */
function getAPIBase() {
  // If running on port 3000 (docker frontend), use /api proxy
  if (window.location.port === "3000") {
    return "/api";
  }
  // If running locally on different port, directly call backend
  if (
    window.location.hostname === "localhost" ||
    window.location.hostname === "127.0.0.1"
  ) {
    return "http://localhost:8080";
  }
  // Production - use same origin
  return "";
}

/**
 * Decode a JWT token and return the payload
 * @param {string} token - JWT token
 * @returns {object} Decoded payload with readable timestamps
 */
function decodeJWT(token) {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) {
      return { error: "Invalid JWT format" };
    }
    const payload = parts[1];
    const decoded = JSON.parse(atob(payload));

    // Convert timestamps to readable dates
    if (decoded.exp) {
      decoded._exp_readable = new Date(decoded.exp * 1000).toISOString();
    }
    if (decoded.iat) {
      decoded._iat_readable = new Date(decoded.iat * 1000).toISOString();
    }
    if (decoded.nbf) {
      decoded._nbf_readable = new Date(decoded.nbf * 1000).toISOString();
    }

    return decoded;
  } catch (err) {
    return { error: "Failed to decode JWT", details: err.message };
  }
}

/**
 * Make an API request with consistent error handling
 * @param {string} endpoint - API endpoint (e.g., "/auth/authorize")
 * @param {object} options - Fetch options
 * @returns {Promise<{data: any, status: number, ok: boolean}>}
 */
async function apiRequest(endpoint, options = {}) {
  const apiBase = getAPIBase();
  const url = `${apiBase}${endpoint}`;

  const headers = {
    "Content-Type": "application/json",
    ...options.headers,
  };

  // Add bearer token if available and not explicitly skipped
  if (!options.skipAuth) {
    const token =
      sessionStorage.getItem("access_token") ||
      localStorage.getItem("access_token");
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const config = {
    ...options,
    headers,
  };

  const res = await fetch(url, config);
  const status = res.status;
  const ok = res.ok;

  // Handle empty responses (204 No Content)
  if (status === 204) {
    return { data: null, status, ok };
  }

  const data = await res.json();

  return { data, status, ok };
}

/**
 * Format expiry time from nanoseconds to human-readable
 * @param {number} expiresIn - Expiry time in nanoseconds
 * @returns {string} Human-readable time (e.g., "15m 30s")
 */
function formatExpiry(expiresIn) {
  const seconds = parseInt(expiresIn) / 1000000000; // Convert nanoseconds to seconds
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.floor(seconds % 60);
  return `${minutes}m ${remainingSeconds}s`;
}

/**
 * Generate a random state value for CSRF protection
 * @returns {string} Random state string
 */
function generateState() {
  return "state-" + Math.random().toString(36).substring(7);
}

/**
 * Get stored tokens from sessionStorage
 * @returns {{accessToken: string|null, idToken: string|null}}
 */
function getStoredTokens() {
  return {
    accessToken: sessionStorage.getItem("access_token"),
    idToken: sessionStorage.getItem("id_token"),
  };
}

/**
 * Store tokens in sessionStorage
 * @param {string} accessToken
 * @param {string} idToken
 */
function storeTokens(accessToken, idToken = null) {
  if (accessToken) {
    sessionStorage.setItem("access_token", accessToken);
  }
  if (idToken) {
    sessionStorage.setItem("id_token", idToken);
  }
}

/**
 * Clear stored tokens
 */
function clearTokens() {
  sessionStorage.removeItem("access_token");
  sessionStorage.removeItem("id_token");
}

/**
 * Build redirect URL with query parameters
 * @param {string} baseUrl - Base redirect URL
 * @param {string} code - Authorization code
 * @param {string} state - State parameter
 * @returns {string} Full redirect URL with params
 */
function buildRedirectUrl(baseUrl, code, state = null) {
  const url = new URL(baseUrl);
  url.searchParams.append("code", code);
  if (state) {
    url.searchParams.append("state", state);
  }
  return url.toString();
}

/**
 * Parse query parameters from current URL
 * @returns {object} Query parameters as key-value pairs
 */
function parseQueryParams() {
  const params = {};
  const urlParams = new URLSearchParams(window.location.search);
  for (const [key, value] of urlParams) {
    params[key] = value;
  }
  return params;
}
