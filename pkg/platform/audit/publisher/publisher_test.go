package publisher

import (
	"context"
	"sync"
	"testing"
	"time"

	id "credo/pkg/domain"
	audit "credo/pkg/platform/audit"
	"credo/pkg/platform/audit/store/memory"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublisher_SyncMode(t *testing.T) {
	store := memory.NewInMemoryStore()
	pub := NewPublisher(store)
	defer pub.Close()

	userID := id.UserID(uuid.New())
	event := audit.Event{
		UserID: userID,
		Action: string(audit.EventUserCreated),
	}

	err := pub.Emit(context.Background(), event)
	require.NoError(t, err)

	events, err := pub.List(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, string(audit.EventUserCreated), events[0].Action)
}

func TestPublisher_AsyncMode(t *testing.T) {
	store := memory.NewInMemoryStore()
	pub := NewPublisher(store, WithAsyncBuffer(10))
	defer pub.Close()

	userID := id.UserID(uuid.New())
	event := audit.Event{
		UserID: userID,
		Action: string(audit.EventConsentGranted),
	}

	err := pub.Emit(context.Background(), event)
	require.NoError(t, err)

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	events, err := pub.List(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, string(audit.EventConsentGranted), events[0].Action)
}

func TestPublisher_AsyncDrainsOnClose(t *testing.T) {
	store := memory.NewInMemoryStore()
	pub := NewPublisher(store, WithAsyncBuffer(100))

	userID := id.UserID(uuid.New())

	// Emit multiple events
	for range 10 {
		event := audit.Event{
			UserID: userID,
			Action: string(audit.EventUserCreated),
		}
		err := pub.Emit(context.Background(), event)
		require.NoError(t, err)
	}

	// Close should drain all events
	pub.Close()

	events, err := store.ListByUser(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, events, 10, "all events should be drained on close")
}

func TestPublisher_BufferFull_DropsEvent(t *testing.T) {
	store := memory.NewInMemoryStore()
	pub := NewPublisher(store, WithAsyncBuffer(1))
	defer pub.Close()

	userID := id.UserID(uuid.New())

	// Fill the buffer with concurrent writes
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event := audit.Event{
				UserID: userID,
				Action: string(audit.EventUserCreated),
			}
			_ = pub.Emit(context.Background(), event)
		}()
	}
	wg.Wait()

	// Some events should have been dropped (buffer size 1)
	// Just verify no panic and publisher still works
}

func TestPublisher_SetsTimestamp(t *testing.T) {
	store := memory.NewInMemoryStore()
	pub := NewPublisher(store)
	defer pub.Close()

	userID := id.UserID(uuid.New())
	event := audit.Event{
		UserID: userID,
		Action: string(audit.EventUserCreated),
		// Timestamp not set
	}

	before := time.Now()
	err := pub.Emit(context.Background(), event)
	require.NoError(t, err)
	after := time.Now()

	events, err := pub.List(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.True(t, !events[0].Timestamp.Before(before), "timestamp should be >= before")
	assert.True(t, !events[0].Timestamp.After(after), "timestamp should be <= after")
}

func TestPublisher_PreservesExistingTimestamp(t *testing.T) {
	store := memory.NewInMemoryStore()
	pub := NewPublisher(store)
	defer pub.Close()

	userID := id.UserID(uuid.New())
	customTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	event := audit.Event{
		UserID:    userID,
		Action:    string(audit.EventUserCreated),
		Timestamp: customTime,
	}

	err := pub.Emit(context.Background(), event)
	require.NoError(t, err)

	events, err := pub.List(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, customTime, events[0].Timestamp)
}

func TestPublisher_ContextCancellation(t *testing.T) {
	store := memory.NewInMemoryStore()
	pub := NewPublisher(store, WithAsyncBuffer(1))
	defer pub.Close()

	// Fill buffer first
	_ = pub.Emit(context.Background(), audit.Event{
		UserID: id.UserID(uuid.New()),
		Action: string(audit.EventUserCreated),
	})

	// Wait for the event to be processed
	time.Sleep(50 * time.Millisecond)

	// Fill buffer again
	_ = pub.Emit(context.Background(), audit.Event{
		UserID: id.UserID(uuid.New()),
		Action: string(audit.EventUserCreated),
	})

	// Try to emit with cancelled context when buffer is full
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pub.Emit(ctx, audit.Event{
		UserID: id.UserID(uuid.New()),
		Action: string(audit.EventUserCreated),
	})

	// Should either succeed (buffer not full) or return context error or buffer full error
	if err != nil {
		assert.True(t, err == context.Canceled || err.Error() == "audit buffer full",
			"expected context.Canceled or buffer full error, got: %v", err)
	}
}

func TestPublisher_MultipleEvents(t *testing.T) {
	store := memory.NewInMemoryStore()
	pub := NewPublisher(store)
	defer pub.Close()

	userID := id.UserID(uuid.New())

	events := []audit.Event{
		{UserID: userID, Action: string(audit.EventUserCreated)},
		{UserID: userID, Action: string(audit.EventSessionCreated)},
		{UserID: userID, Action: string(audit.EventTokenIssued)},
	}

	for _, event := range events {
		err := pub.Emit(context.Background(), event)
		require.NoError(t, err)
	}

	result, err := pub.List(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, result, 3)

	assert.Equal(t, string(audit.EventUserCreated), result[0].Action)
	assert.Equal(t, string(audit.EventSessionCreated), result[1].Action)
	assert.Equal(t, string(audit.EventTokenIssued), result[2].Action)
}

func TestPublisher_DifferentUsers(t *testing.T) {
	store := memory.NewInMemoryStore()
	pub := NewPublisher(store)
	defer pub.Close()

	userID1 := id.UserID(uuid.New())
	userID2 := id.UserID(uuid.New())

	err := pub.Emit(context.Background(), audit.Event{
		UserID: userID1,
		Action: string(audit.EventUserCreated),
	})
	require.NoError(t, err)

	err = pub.Emit(context.Background(), audit.Event{
		UserID: userID2,
		Action: string(audit.EventConsentGranted),
	})
	require.NoError(t, err)

	events1, err := pub.List(context.Background(), userID1)
	require.NoError(t, err)
	require.Len(t, events1, 1)
	assert.Equal(t, string(audit.EventUserCreated), events1[0].Action)

	events2, err := pub.List(context.Background(), userID2)
	require.NoError(t, err)
	require.Len(t, events2, 1)
	assert.Equal(t, string(audit.EventConsentGranted), events2[0].Action)
}
