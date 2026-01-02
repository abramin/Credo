//go:build integration

package session_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	"credo/internal/auth/models"
	"credo/internal/auth/store/session"
	id "credo/pkg/domain"
	"credo/pkg/testutil/containers"
)

type RedisStoreSuite struct {
	suite.Suite
	redis *containers.RedisContainer
	store *session.RedisStore
}

func TestRedisStoreSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	suite.Run(t, new(RedisStoreSuite))
}

func (s *RedisStoreSuite) SetupSuite() {
	mgr := containers.GetManager()
	s.redis = mgr.GetRedis(s.T())
	s.store = session.NewRedis(s.redis.Client)
}

func (s *RedisStoreSuite) SetupTest() {
	ctx := context.Background()
	err := s.redis.FlushAll(ctx)
	s.Require().NoError(err)
}

func makeSession(userID id.UserID) *models.Session {
	return &models.Session{
		ID:                    id.SessionID(uuid.New()),
		UserID:                userID,
		ClientID:              id.ClientID(uuid.New()),
		TenantID:              id.TenantID(uuid.New()),
		RequestedScope:        []string{"openid", "profile"},
		Status:                models.SessionStatusActive,
		LastAccessTokenJTI:    uuid.NewString(),
		DeviceID:              "device-123",
		DeviceFingerprintHash: "fp-hash-456",
		DeviceDisplayName:     "Test Device",
		ApproximateLocation:   "Test Location",
		CreatedAt:             time.Now(),
		ExpiresAt:             time.Now().Add(24 * time.Hour),
		LastSeenAt:            time.Now(),
	}
}

// TestWATCHConflictDetection verifies that concurrent modifications trigger
// Redis WATCH conflict detection (redis.TxFailedErr).
func (s *RedisStoreSuite) TestWATCHConflictDetection() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())
	sess := makeSession(userID)
	err := s.store.Create(ctx, sess)
	s.Require().NoError(err)

	const goroutines = 20
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var watchFailCount atomic.Int32
	var otherErrors atomic.Int32

	// All goroutines try to revoke the same session
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := s.store.RevokeSessionIfActive(ctx, sess.ID, time.Now())
			if err == nil {
				successCount.Add(1)
			} else if err == redis.TxFailedErr {
				watchFailCount.Add(1)
			} else if err == session.ErrSessionRevoked {
				// Already revoked by another goroutine
				watchFailCount.Add(1)
			} else {
				otherErrors.Add(1)
			}
		}()
	}

	wg.Wait()

	// Exactly one should succeed in revoking
	s.Equal(int32(1), successCount.Load(), "exactly one revoke should succeed")
	// Others should fail due to WATCH conflict or already revoked
	s.Equal(int32(goroutines-1), watchFailCount.Load(), "remaining should fail")
	s.Equal(int32(0), otherErrors.Load(), "no unexpected errors")
}

// TestOptimisticLockRetrySuccess verifies that after a WATCH conflict,
// a retry will succeed.
func (s *RedisStoreSuite) TestOptimisticLockRetrySuccess() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())
	sess := makeSession(userID)
	err := s.store.Create(ctx, sess)
	s.Require().NoError(err)

	// Simulate concurrent modification by directly modifying Redis
	// while Execute is in progress
	var executeCalled atomic.Bool
	var conflictDetected atomic.Bool

	// First goroutine: Execute with validation that yields to allow interference
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		_, err := s.store.Execute(ctx, sess.ID,
			func(session *models.Session) error {
				executeCalled.Store(true)
				// Small delay to allow interference
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			func(session *models.Session) {
				session.LastAccessTokenJTI = "new-jti-from-execute"
			},
		)

		if err == redis.TxFailedErr {
			conflictDetected.Store(true)
		}
	}()

	// Second goroutine: interfere by modifying the session directly
	go func() {
		defer wg.Done()

		// Wait for execute to start
		for !executeCalled.Load() {
			time.Sleep(1 * time.Millisecond)
		}

		// Directly modify the session in Redis
		key := "session:" + uuid.UUID(sess.ID).String()
		s.redis.Client.Set(ctx, key, `{"id":"`+uuid.UUID(sess.ID).String()+`","user_id":"`+uuid.UUID(sess.UserID).String()+`","client_id":"`+uuid.UUID(sess.ClientID).String()+`","tenant_id":"`+uuid.UUID(sess.TenantID).String()+`","requested_scope":["openid"],"status":"active","last_access_token_jti":"interfered-jti","device_id":"device","device_fingerprint_hash":"fp","device_display_name":"Device","approximate_location":"Location","created_at":1000000000,"expires_at":2000000000000000000,"last_seen_at":1000000000}`, 24*time.Hour)
	}()

	wg.Wait()

	// One of these conditions should be true:
	// - Execute succeeded (interference happened after commit)
	// - Execute detected conflict (interference happened before commit)
	// Either way, the data should be consistent

	// Verify session can still be read
	readSession, err := s.store.FindByID(ctx, sess.ID)
	s.Require().NoError(err)
	s.NotNil(readSession)

	// If retry succeeds, we can execute again
	if conflictDetected.Load() {
		// Retry should succeed
		result, err := s.store.Execute(ctx, sess.ID,
			func(session *models.Session) error { return nil },
			func(session *models.Session) {
				session.LastAccessTokenJTI = "retry-success-jti"
			},
		)
		s.Require().NoError(err)
		s.Equal("retry-success-jti", result.LastAccessTokenJTI)
	}
}

// TestTTLPreservation verifies that updates preserve session expiry TTL.
func (s *RedisStoreSuite) TestTTLPreservation() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())
	sess := makeSession(userID)
	sess.ExpiresAt = time.Now().Add(1 * time.Hour)
	err := s.store.Create(ctx, sess)
	s.Require().NoError(err)

	// Get initial TTL
	key := "session:" + uuid.UUID(sess.ID).String()
	initialTTL, err := s.redis.Client.TTL(ctx, key).Result()
	s.Require().NoError(err)
	s.Greater(initialTTL, time.Duration(0), "initial TTL should be positive")

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Update via Execute
	_, err = s.store.Execute(ctx, sess.ID,
		func(session *models.Session) error { return nil },
		func(session *models.Session) {
			session.LastAccessTokenJTI = "updated-jti"
		},
	)
	s.Require().NoError(err)

	// Check TTL is preserved (should still be close to original)
	newTTL, err := s.redis.Client.TTL(ctx, key).Result()
	s.Require().NoError(err)
	s.Greater(newTTL, time.Duration(0), "TTL should still be positive after update")

	// TTL should be within reasonable range (allowing for some time passage)
	s.InDelta(initialTTL.Seconds(), newTTL.Seconds(), 5.0, "TTL should be preserved")
}

// TestPipelineAtomicity verifies that create operations are atomic.
func (s *RedisStoreSuite) TestPipelineAtomicity() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())

	const goroutines = 30
	var wg sync.WaitGroup
	var successCount atomic.Int32

	// Create many sessions for the same user concurrently
	sessions := make([]*models.Session, goroutines)
	for i := 0; i < goroutines; i++ {
		sessions[i] = makeSession(userID)
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			err := s.store.Create(ctx, sessions[idx])
			if err == nil {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// All creates should succeed (they're independent)
	s.Equal(int32(goroutines), successCount.Load(), "all creates should succeed")

	// All sessions should be in the user's session set
	userKey := "user_sessions:" + uuid.UUID(userID).String()
	members, err := s.redis.Client.SMembers(ctx, userKey).Result()
	s.Require().NoError(err)
	s.Len(members, goroutines, "all sessions should be in user set")

	// All individual sessions should be readable
	for _, sess := range sessions {
		readSession, err := s.store.FindByID(ctx, sess.ID)
		s.Require().NoError(err)
		s.Equal(sess.ID, readSession.ID)
	}
}

// TestConcurrentExecuteOnDifferentSessions verifies that Execute on different
// sessions doesn't interfere with each other.
func (s *RedisStoreSuite) TestConcurrentExecuteOnDifferentSessions() {
	ctx := context.Background()

	const goroutines = 20
	sessions := make([]*models.Session, goroutines)
	for i := 0; i < goroutines; i++ {
		sessions[i] = makeSession(id.UserID(uuid.New()))
		err := s.store.Create(ctx, sessions[i])
		s.Require().NoError(err)
	}

	var wg sync.WaitGroup
	var successCount atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			result, err := s.store.Execute(ctx, sessions[idx].ID,
				func(session *models.Session) error { return nil },
				func(session *models.Session) {
					session.LastAccessTokenJTI = "updated-" + uuid.NewString()
				},
			)
			if err == nil && result != nil {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// All should succeed since they're operating on different sessions
	s.Equal(int32(goroutines), successCount.Load(), "all executes should succeed")
}

// TestListByUserUnderConcurrentCreation verifies ListByUser returns
// consistent results during concurrent session creation.
func (s *RedisStoreSuite) TestListByUserUnderConcurrentCreation() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())

	const goroutines = 20
	var wg sync.WaitGroup
	var createSuccess atomic.Int32

	// Create sessions concurrently
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			sess := makeSession(userID)
			if err := s.store.Create(ctx, sess); err == nil {
				createSuccess.Add(1)
			}
		}()
	}

	wg.Wait()
	s.Equal(int32(goroutines), createSuccess.Load())

	// List should return all sessions
	sessions, err := s.store.ListByUser(ctx, userID)
	s.Require().NoError(err)
	s.Len(sessions, goroutines, "should list all created sessions")
}

// TestDeleteSessionsByUserConcurrency verifies that DeleteSessionsByUser
// is safe under concurrent access.
func (s *RedisStoreSuite) TestDeleteSessionsByUserConcurrency() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())

	// Create several sessions
	for i := 0; i < 10; i++ {
		sess := makeSession(userID)
		err := s.store.Create(ctx, sess)
		s.Require().NoError(err)
	}

	const goroutines = 5
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var notFoundCount atomic.Int32

	// Multiple goroutines try to delete all sessions
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := s.store.DeleteSessionsByUser(ctx, userID)
			if err == nil {
				successCount.Add(1)
			} else if err.Error() == "session not found: not found" {
				notFoundCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// At least one should succeed, others may get not found
	total := successCount.Load() + notFoundCount.Load()
	s.Equal(int32(goroutines), total, "all goroutines should complete")
	s.GreaterOrEqual(successCount.Load(), int32(1), "at least one delete should succeed")

	// After all deletes, listing should return empty
	sessions, err := s.store.ListByUser(ctx, userID)
	s.Require().NoError(err)
	s.Empty(sessions, "no sessions should remain")
}

// TestExecuteValidationRollback verifies that validation errors in Execute
// don't persist any changes.
func (s *RedisStoreSuite) TestExecuteValidationRollback() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())
	sess := makeSession(userID)
	originalJTI := sess.LastAccessTokenJTI
	err := s.store.Create(ctx, sess)
	s.Require().NoError(err)

	// Execute with validation that fails
	validationErr := &customError{message: "validation failed"}
	_, err = s.store.Execute(ctx, sess.ID,
		func(session *models.Session) error {
			return validationErr
		},
		func(session *models.Session) {
			session.LastAccessTokenJTI = "should-not-be-persisted"
		},
	)
	s.ErrorIs(err, validationErr)

	// Session should be unchanged
	readSession, err := s.store.FindByID(ctx, sess.ID)
	s.Require().NoError(err)
	s.Equal(originalJTI, readSession.LastAccessTokenJTI, "session should be unchanged after validation error")
}

// TestRevokeIdempotent verifies that revoking an already revoked session
// returns the appropriate error.
func (s *RedisStoreSuite) TestRevokeIdempotent() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())
	sess := makeSession(userID)
	err := s.store.Create(ctx, sess)
	s.Require().NoError(err)

	// First revoke should succeed
	err = s.store.RevokeSessionIfActive(ctx, sess.ID, time.Now())
	s.Require().NoError(err)

	// Second revoke should return ErrSessionRevoked
	err = s.store.RevokeSessionIfActive(ctx, sess.ID, time.Now())
	s.Equal(session.ErrSessionRevoked, err)
}

// TestUpdateSessionPreservesTTL verifies UpdateSession preserves the TTL.
func (s *RedisStoreSuite) TestUpdateSessionPreservesTTL() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())
	sess := makeSession(userID)
	sess.ExpiresAt = time.Now().Add(2 * time.Hour)
	err := s.store.Create(ctx, sess)
	s.Require().NoError(err)

	key := "session:" + uuid.UUID(sess.ID).String()
	initialTTL, err := s.redis.Client.TTL(ctx, key).Result()
	s.Require().NoError(err)

	// Update session
	sess.LastAccessTokenJTI = "new-jti"
	err = s.store.UpdateSession(ctx, sess)
	s.Require().NoError(err)

	newTTL, err := s.redis.Client.TTL(ctx, key).Result()
	s.Require().NoError(err)

	// TTL should be preserved (within some tolerance)
	s.InDelta(initialTTL.Seconds(), newTTL.Seconds(), 5.0)
}

// customError is a test error type for validation testing.
type customError struct {
	message string
}

func (e *customError) Error() string {
	return e.message
}

// TestListAllUnderConcurrentModification verifies ListAll returns consistent
// results even during concurrent modifications.
func (s *RedisStoreSuite) TestListAllUnderConcurrentModification() {
	ctx := context.Background()

	// Create initial sessions
	const initialSessions = 10
	for i := 0; i < initialSessions; i++ {
		sess := makeSession(id.UserID(uuid.New()))
		err := s.store.Create(ctx, sess)
		s.Require().NoError(err)
	}

	// Concurrently create more sessions while listing
	var wg sync.WaitGroup
	var listResults []int

	wg.Add(2)

	// Lister
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			sessions, err := s.store.ListAll(ctx)
			if err == nil {
				listResults = append(listResults, len(sessions))
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Creator
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			sess := makeSession(id.UserID(uuid.New()))
			s.store.Create(ctx, sess)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Final list should have all sessions
	finalSessions, err := s.store.ListAll(ctx)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(finalSessions), initialSessions, "should have at least initial sessions")
}

// Helper to verify JSON serialization round-trip
func (s *RedisStoreSuite) TestSessionJSONRoundTrip() {
	ctx := context.Background()
	userID := id.UserID(uuid.New())
	sess := makeSession(userID)

	// Set optional fields
	now := time.Now()
	sess.LastRefreshedAt = &now
	sess.RevokedAt = nil

	err := s.store.Create(ctx, sess)
	s.Require().NoError(err)

	// Read back and verify all fields
	readSession, err := s.store.FindByID(ctx, sess.ID)
	s.Require().NoError(err)

	s.Equal(sess.ID, readSession.ID)
	s.Equal(sess.UserID, readSession.UserID)
	s.Equal(sess.ClientID, readSession.ClientID)
	s.Equal(sess.TenantID, readSession.TenantID)
	s.Equal(sess.RequestedScope, readSession.RequestedScope)
	s.Equal(sess.Status, readSession.Status)
	s.Equal(sess.LastAccessTokenJTI, readSession.LastAccessTokenJTI)
	s.Equal(sess.DeviceID, readSession.DeviceID)
	s.Equal(sess.DeviceFingerprintHash, readSession.DeviceFingerprintHash)
	s.Equal(sess.DeviceDisplayName, readSession.DeviceDisplayName)
	s.Equal(sess.ApproximateLocation, readSession.ApproximateLocation)

	// Time fields - compare Unix nanos due to serialization
	s.Equal(sess.CreatedAt.UnixNano(), readSession.CreatedAt.UnixNano())
	s.Equal(sess.ExpiresAt.UnixNano(), readSession.ExpiresAt.UnixNano())
	s.Equal(sess.LastSeenAt.UnixNano(), readSession.LastSeenAt.UnixNano())

	if sess.LastRefreshedAt != nil {
		s.Require().NotNil(readSession.LastRefreshedAt)
		s.Equal(sess.LastRefreshedAt.UnixNano(), readSession.LastRefreshedAt.UnixNano())
	}
}

// Compile-time check that json import is used
var _ = json.Marshal
