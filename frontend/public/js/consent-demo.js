// Consent Demo Component
// Implements the consent management UI for PRD-002
// Assumes backend is always available

const CONSENT_API_BASE_URL = (() => {
    if (window.location.port === '3000') {
        return '/api';
    }
    if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
        return 'http://localhost:8080';
    }
    return '';
})();

const consentApiUrl = (endpoint) => `${CONSENT_API_BASE_URL}${endpoint}`;

document.addEventListener('alpine:init', () => {
    Alpine.data('consentDemo', () => ({
        // State
        loading: false,
        purposesLoading: false,
        consentsLoading: false,
        runnerLoading: false,
        error: null,
        success: null,
        activeTab: 'consents',

        // Authentication
        isAuthenticated: false,
        accessToken: null,

        // Demo users
        users: [
            { id: 'alice@example.com', name: 'Alice Smith' },
            { id: 'ahmed@example.com', name: 'Ahmed Tawfiq' },
            { id: 'diego@example.com', name: 'Diego Garcia' },
            { id: 'demo@example.com', name: 'Demo User' }
        ],
        currentUser: '',

        // Data - purposes are hardcoded based on backend validation
        purposes: [
            {
                id: 'login',
                name: 'User Login & Authentication',
                description: 'Allow authentication and session management',
                default_ttl_hours: 8760 // 365 days
            },
            {
                id: 'vc_issuance',
                name: 'Verifiable Credential Issuance',
                description: 'Allow issuing verifiable credentials on your behalf',
                default_ttl_hours: 8760 // 365 days
            },
            {
                id: 'registry_check',
                name: 'Registry Background Check',
                description: 'Perform background checks via registry integration',
                default_ttl_hours: 2160 // 90 days
            },
            {
                id: 'decision_evaluation',
                name: 'Decision Engine Evaluation',
                description: 'Run authorization decisions through policy engine',
                default_ttl_hours: 4320 // 180 days
            },
            {
                id: 'invalid_purpose',
                name: '⚠️ Invalid Purpose (Demo)',
                description: 'This purpose is not valid in the backend. Try granting it to see error handling.',
                default_ttl_hours: 720,
                isInvalid: true
            }
        ],
        consents: [],
        selectedScenario: 'valid',
        runnerNationalId: 'CITIZEN123456',
        runnerPurpose: 'age_verification',
        lastCredentialId: '',
        runnerSteps: [
            {
                id: 'citizen',
                name: 'Citizen Registry',
                description: 'POST /registry/citizen'
            },
            {
                id: 'sanctions',
                name: 'Sanctions Check',
                description: 'POST /registry/sanctions'
            },
            {
                id: 'vc_issue',
                name: 'Issue Credential',
                description: 'POST /vc/issue'
            },
            {
                id: 'vc_verify',
                name: 'Verify Credential',
                description: 'POST /vc/verify'
            },
            {
                id: 'decision',
                name: 'Evaluate Decision',
                description: 'POST /decision/evaluate'
            }
        ],
        runnerScenarios: [
            {
                id: 'valid',
                label: 'Valid',
                nationalId: 'CITIZEN123456',
                hint: 'Expected: valid citizen'
            },
            {
                id: 'sanctioned',
                label: 'Sanctioned',
                nationalId: 'SANCTIONED999',
                hint: 'Expected: sanctions listed'
            },
            {
                id: 'invalid',
                label: 'Invalid',
                nationalId: 'INVALID999',
                hint: 'Expected: invalid citizen'
            },
            {
                id: 'underage',
                label: 'Underage',
                nationalId: 'UNDERAGE01',
                hint: 'Depends on registry data'
            }
        ],
        runnerResults: {
            citizen: null,
            sanctions: null,
            vc_issue: null,
            vc_verify: null,
            decision: null
        },
        runnerExpanded: {
            citizen: false,
            sanctions: false,
            vc_issue: false,
            vc_verify: false,
            decision: false
        },

        // Initialization
        init() {
            // Check if we have a token from OAuth flow
            this.checkForToken();
            // Auto-dismiss notifications after 5 seconds
            this.$watch('error', () => {
                if (this.error) {
                    setTimeout(() => { this.error = null; }, 5000);
                }
            });
            this.$watch('success', () => {
                if (this.success) {
                    setTimeout(() => { this.success = null; }, 5000);
                }
            });
        },

        applyScenario(scenario) {
            this.selectedScenario = scenario.id;
            this.runnerNationalId = scenario.nationalId;
        },

        resetRunner() {
            this.runnerResults = {
                citizen: null,
                sanctions: null,
                vc_issue: null,
                vc_verify: null,
                decision: null
            };
            this.runnerExpanded = {
                citizen: false,
                sanctions: false,
                vc_issue: false,
                vc_verify: false,
                decision: false
            };
            this.lastCredentialId = '';
        },

        async runAll() {
            if (!this.runnerNationalId) {
                this.error = 'Please enter a National ID before running the pipeline.';
                return;
            }

            if (!this.isAuthenticated || !this.accessToken) {
                this.error = 'Not authenticated. Please get an access token first.';
                return;
            }

            this.runnerLoading = true;
            this.error = null;
            this.success = null;
            this.resetRunner();

            await this.runCitizen();
            await this.runSanctions();
            await this.runIssueVC();
            await this.runVerifyVC();
            await this.runDecision();

            this.runnerLoading = false;
        },

        async runStep(stepId) {
            if (!this.runnerNationalId) {
                this.error = 'Please enter a National ID before running a step.';
                return;
            }

            if (!this.isAuthenticated || !this.accessToken) {
                this.error = 'Not authenticated. Please get an access token first.';
                return;
            }

            this.runnerLoading = true;
            try {
                switch (stepId) {
                    case 'citizen':
                        await this.runCitizen();
                        break;
                    case 'sanctions':
                        await this.runSanctions();
                        break;
                    case 'vc_issue':
                        await this.runIssueVC();
                        break;
                    case 'vc_verify':
                        await this.runVerifyVC();
                        break;
                    case 'decision':
                        await this.runDecision();
                        break;
                    default:
                        break;
                }
            } finally {
                this.runnerLoading = false;
            }
        },

        async runCitizen() {
            await this.runnerRequest('citizen', '/registry/citizen', {
                national_id: this.runnerNationalId
            });
        },

        async runSanctions() {
            await this.runnerRequest('sanctions', '/registry/sanctions', {
                national_id: this.runnerNationalId
            });
        },

        async runIssueVC() {
            const result = await this.runnerRequest('vc_issue', '/vc/issue', {
                type: 'AgeOver18',
                national_id: this.runnerNationalId
            });

            if (result && result.state === 'success') {
                const body = result.body || {};
                this.lastCredentialId = body.credential_id || body.credentialId || body.ID || '';
            }
        },

        async runVerifyVC() {
            if (!this.lastCredentialId) {
                this.runnerResults.vc_verify = {
                    state: 'skipped',
                    status: null,
                    durationMs: 0,
                    body: null,
                    error: 'No credential_id available. Issue a credential first.'
                };
                return;
            }

            await this.runnerRequest('vc_verify', '/vc/verify', {
                credential_id: this.lastCredentialId
            });
        },

        async runDecision() {
            await this.runnerRequest('decision', '/decision/evaluate', {
                purpose: this.runnerPurpose,
                context: {
                    national_id: this.runnerNationalId
                }
            });
        },

        async runnerRequest(stepId, endpoint, payload) {
            const start = performance.now();
            try {
                const response = await fetch(consentApiUrl(endpoint), {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + this.accessToken
                    },
                    body: JSON.stringify(payload)
                });

                const durationMs = Math.round(performance.now() - start);
                const contentType = response.headers.get('content-type') || '';
                let body = null;

                if (response.status !== 204) {
                    if (contentType.includes('application/json')) {
                        body = await response.json();
                    } else {
                        body = await response.text();
                    }
                }

                const result = {
                    state: response.ok ? 'success' : 'error',
                    status: response.status,
                    durationMs,
                    body,
                    error: response.ok ? null : (body && (body.error_description || body.error || body.message)) || response.statusText
                };

                this.runnerResults[stepId] = result;
                return result;
            } catch (err) {
                const durationMs = Math.round(performance.now() - start);
                const result = {
                    state: 'error',
                    status: 0,
                    durationMs,
                    body: null,
                    error: err.message || 'Network error'
                };
                this.runnerResults[stepId] = result;
                return result;
            }
        },

        toggleRunnerDetails(stepId) {
            this.runnerExpanded[stepId] = !this.runnerExpanded[stepId];
        },

        runnerCardClass(stepId) {
            const result = this.runnerResults[stepId];
            if (!result) return 'border-gray-200 bg-white';
            if (result.state === 'success') return 'border-green-200 bg-green-50';
            if (result.state === 'error') return 'border-red-200 bg-red-50';
            if (result.state === 'skipped') return 'border-yellow-200 bg-yellow-50';
            return 'border-gray-200 bg-white';
        },

        runnerStatusClass(stepId) {
            const result = this.runnerResults[stepId];
            if (!result) return '';
            if (result.state === 'success') return 'pass';
            if (result.state === 'error') return 'fail';
            if (result.state === 'skipped') return 'pending';
            return '';
        },

        runnerStatusText(stepId) {
            const result = this.runnerResults[stepId];
            if (!result) return 'Not Run';
            if (result.state === 'success') return 'OK';
            if (result.state === 'error') return 'Error';
            if (result.state === 'skipped') return 'Skipped';
            return 'Unknown';
        },

        formatRunnerBody(stepId) {
            const result = this.runnerResults[stepId];
            if (!result) return '';
            const payload = result.body || { error: result.error || 'No response body' };
            if (typeof payload === 'string') {
                return payload;
            }
            try {
                return JSON.stringify(payload, null, 2);
            } catch (err) {
                return String(payload);
            }
        },
        // Check if OAuth token is available in URL or sessionStorage
        checkForToken() {
            // Look for token in URL query parameter (e.g., ?token=xxx)
            const urlParams = new URLSearchParams(window.location.search);
            const tokenParam = urlParams.get('token');
            if (tokenParam) {
                this.accessToken = tokenParam;
                this.isAuthenticated = true;
                sessionStorage.setItem('access_token', tokenParam);
                // Clean up URL
                window.history.replaceState({}, document.title, window.location.pathname);
                this.success = '✅ Authenticated with provided token!';
                return;
            }

            // Look for token in URL hash (from OAuth redirect)
            const hash = window.location.hash;
            if (hash.includes('access_token=')) {
                const match = hash.match(/access_token=([^&]+)/);
                if (match) {
                    this.accessToken = match[1];
                    this.isAuthenticated = true;
                    sessionStorage.setItem('access_token', match[1]);
                    // Clean up URL
                    window.history.replaceState({}, document.title, window.location.pathname);
                    this.success = '✅ Authenticated! You can now test consent endpoints.';
                    return;
                }
            }

            // Check sessionStorage for token
            const storedToken = sessionStorage.getItem('access_token');
            if (storedToken) {
                this.accessToken = storedToken;
                this.isAuthenticated = true;
                return;
            }

            // Check for auto-auth via demo mode (generate a test token)
            if (urlParams.get('demo') === 'true' || urlParams.get('auto') === 'true') {
                // Generate a test token for demo purposes
                // In a real scenario, this would come from the OAuth flow
                this.accessToken = 'demo_token_' + Math.random().toString(36).substring(2, 15);
                this.isAuthenticated = true;
                sessionStorage.setItem('access_token', this.accessToken);
                this.success = '✅ Auto-authenticated in demo mode! (Using test token)';
                return;
            }

            // Not authenticated
        },

        // Start OAuth flow to get token
        startOAuthFlow() {
            // Store clean return URL (without hash or query params)
            const returnUrl = window.location.origin + window.location.pathname;
            sessionStorage.setItem('consent-demo-return', returnUrl);
            // Redirect to OAuth demo to get token
            window.location.href = '/demo.html?return=consent-demo.html';
        },

        // User selection
        async switchUser() {
            this.error = null;
            this.success = null;
            this.consents = [];
            
            if (!this.currentUser) {
                return;
            }

            // Load consents for the selected user
            await this.loadUserConsents();
        },

        // Load user consents from backend
        async loadUserConsents() {
            if (!this.currentUser) return;

            this.consentsLoading = true;
            this.error = null;

            try {
                // Fetch from backend: GET /auth/consent
                const response = await fetch(consentApiUrl('/auth/consent'), {
                    method: 'GET',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + this.accessToken
                    }
                });

                if (!response.ok) {
                    if (response.status === 401) {
                        throw new Error('Unauthorized - token may be expired. Please authenticate again.');
                    }
                    if (response.status === 404) {
                        throw new Error('Backend endpoint not found. Is the server running?');
                    }
                    // Try to get error message from response
                    const contentType = response.headers.get('content-type');
                    if (contentType && contentType.includes('application/json')) {
                        const errorData = await response.json();
                        throw new Error(errorData.error || errorData.message || `HTTP ${response.status}: ${response.statusText}`);
                    }
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                const contentType = response.headers.get('content-type');
                if (!contentType || !contentType.includes('application/json')) {
                    throw new Error(`Backend returned ${contentType || 'unknown content'} instead of JSON. Status: ${response.status}`);
                }

                const data = await response.json();
                // Transform backend format to UI format
                this.consents = (data.consents || data.Consents || []).map(c => ({
                    id: c.id,
                    purpose: c.purpose,
                    status: c.status,
                    granted_at: c.granted_at,
                    expires_at: c.expires_at,
                    revoked_at: c.revoked_at
                }));
            } catch (err) {
                console.error('Failed to load consents:', err);
                this.error = `Failed to load consents: ${err.message}`;
                this.consents = [];
            } finally {
                this.consentsLoading = false;
            }
        },

        // Grant consent via backend
        async grantConsent(purposeId) {
            if (!this.currentUser) {
                this.error = 'Please select a user first';
                return;
            }

            if (!this.isAuthenticated || !this.accessToken) {
                this.error = 'Not authenticated. Please get an access token first.';
                return;
            }

            this.loading = true;
            this.error = null;
            this.success = null;

            const purpose = this.purposes.find(p => p.id === purposeId);
            const purposeName = purpose ? purpose.name : purposeId;

            try {
                // Check if trying to grant invalid purpose
                if (purpose && purpose.isInvalid) {
                    throw new Error(`Invalid purpose: "${purposeId}" is not valid. This demonstrates backend validation - invalid purposes will be rejected.`);
                }

                const payload = {
                    purposes: [purposeId]  // Backend expects array
                };

                // Call backend: POST /auth/consent
                const response = await fetch(consentApiUrl('/auth/consent'), {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + this.accessToken
                    },
                    body: JSON.stringify(payload)
                });

                if (!response.ok) {
                    const errorData = await response.json().catch(() => ({}));
                    throw new Error(errorData.error || errorData.message || `HTTP ${response.status}: ${response.statusText}`);
                }

                const data = await response.json();
                this.success = `✅ ${data.message || 'Consent granted for ' + purposeName}`;

                // Reload consents after granting
                await this.loadUserConsents();
            } catch (err) {
                console.error('Failed to grant consent:', err);
                // Surface backend validation errors directly to the UI
                this.error = err.message || 'Failed to grant consent';
                // Bring the error banner into view
                window.scrollTo({ top: 0, behavior: 'smooth' });
            } finally {
                this.loading = false;
            }
        },

        // Revoke consent via backend
        async revokeConsent(purposeId) {
            if (!this.currentUser) {
                this.error = 'Please select a user first';
                return;
            }

            if (!this.isAuthenticated || !this.accessToken) {
                this.error = 'Not authenticated. Please get an access token first.';
                return;
            }

            if (!confirm(`Are you sure you want to revoke consent for "${purposeId}"?`)) {
                return;
            }

            this.loading = true;
            this.error = null;
            this.success = null;

            const purpose = this.purposes.find(p => p.id === purposeId);
            const purposeName = purpose ? purpose.name : purposeId;

            try {
                const payload = {
                    purposes: [purposeId]  // Backend expects array
                };

                // Call backend: POST /auth/consent/revoke
                const response = await fetch(consentApiUrl('/auth/consent/revoke'), {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': 'Bearer ' + this.accessToken
                    },
                    body: JSON.stringify(payload)
                });

                if (!response.ok) {
                    const errorData = await response.json().catch(() => ({}));
                    throw new Error(errorData.error || errorData.message || `HTTP ${response.status}: ${response.statusText}`);
                }

                const data = await response.json();
                this.success = `✅ ${data.message || 'Consent revoked for ' + purposeName}`;

                // Reload consents after revoking
                await this.loadUserConsents();
            } catch (err) {
                console.error('Failed to revoke consent:', err);
                this.error = `Failed to revoke consent: ${err.message}`;
            } finally {
                this.loading = false;
            }
        },

        // Helper: Check if purpose is already granted
        isGranted(purposeId) {
            return this.consents.some(c => c.purpose === purposeId && c.status === 'active');
        },

        // Helper: Get status badge class
        getStatusClass(status) {
            switch (status) {
                case 'active':
                    return 'pass';
                case 'revoked':
                    return 'fail';
                case 'expired':
                    return 'pending';
                default:
                    return '';
            }
        },

        // Helper: Format date/time
        formatDateTime(isoString) {
            if (!isoString) return 'N/A';
            const date = new Date(isoString);
            return date.toLocaleString('en-US', {
                year: 'numeric',
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        },

        // Helper: Format duration in hours to human-readable
        formatDuration(hours) {
            if (!hours) return 'N/A';
            if (hours < 24) return `${hours}h`;
            const days = Math.floor(hours / 24);
            if (days < 30) return `${days}d`;
            const months = Math.floor(days / 30);
            if (months < 12) return `${months}mo`;
            const years = Math.floor(months / 12);
            return `${years}y`;
        },

        // Helper: Calculate time remaining
        getTimeRemaining(expiresAt) {
            if (!expiresAt) return '';
            const now = new Date();
            const expiry = new Date(expiresAt);
            const diff = expiry - now;

            if (diff < 0) return 'Expired';

            const days = Math.floor(diff / (1000 * 60 * 60 * 24));
            const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));

            if (days > 0) return `${days}d ${hours}h remaining`;
            if (hours > 0) return `${hours}h remaining`;
            const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
            return `${minutes}m remaining`;
        }
    }));
});
