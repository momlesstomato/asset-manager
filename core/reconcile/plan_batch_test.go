package reconcile

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestApplyPlan_UsesBatchDeletion tests that ApplyPlan uses batch methods when available.
func TestApplyPlan_UsesBatchDeletion(t *testing.T) {
	mutator := &mockBatchMutator{
		mockMutator: mockMutator{
			mockAdapter: mockAdapter{},
		},
		batchDBCalls:       make([][]string, 0),
		batchGamedataCalls: make([][]string, 0),
		batchStorageCalls:  make([][]string, 0),
	}

	spec := &Spec{
		Adapter:  mutator,
		CacheTTL: 0,
	}

	plan := &ReconcilePlan{
		Actions: []Action{
			{Type: ActionDeleteDB, Key: "1"},
			{Type: ActionDeleteDB, Key: "2"},
			{Type: ActionDeleteDB, Key: "3"},
			{Type: ActionDeleteGamedata, Key: "10"},
			{Type: ActionDeleteGamedata, Key: "11"},
			{Type: ActionDeleteStorage, Key: "20"},
			{Type: ActionDeleteStorage, Key: "21"},
			{Type: ActionDeleteStorage, Key: "22"},
		},
	}

	opts := ReconcileOptions{
		Confirmed: true,
		DryRun:    false,
	}

	executed, err := ApplyPlan(context.Background(), spec, nil, nil, "", plan, opts)
	assert.NoError(t, err)
	assert.Equal(t, 8, executed)

	// Verify batch methods were called, NOT individual methods
	assert.Len(t, mutator.batchDBCalls, 1, "Should use batch DB delete")
	assert.Equal(t, []string{"1", "2", "3"}, mutator.batchDBCalls[0])
	assert.Len(t, mutator.deletedDB, 0, "Should NOT use individual DB delete")

	assert.Len(t, mutator.batchGamedataCalls, 1, "Should use batch gamedata delete")
	assert.Equal(t, []string{"10", "11"}, mutator.batchGamedataCalls[0])
	assert.Len(t, mutator.deletedGamedata, 0, "Should NOT use individual gamedata delete")

	assert.Len(t, mutator.batchStorageCalls, 1, "Should use batch storage delete")
	assert.Equal(t, []string{"20", "21", "22"}, mutator.batchStorageCalls[0])
	assert.Len(t, mutator.deletedStorage, 0, "Should NOT use individual storage delete")
}

// TestApplyPlan_FallbackToSequential tests fallback when batch methods not available.
func TestApplyPlan_FallbackToSequential(t *testing.T) {
	// Use mockMutator without batch methods
	mutator := &mockMutator{
		mockAdapter:     mockAdapter{},
		deletedDB:       make([]string, 0),
		deletedGamedata: make([]string, 0),
		deletedStorage:  make([]string, 0),
		synced:          make([]string, 0),
	}

	spec := &Spec{
		Adapter:  mutator,
		CacheTTL: 0,
	}

	plan := &ReconcilePlan{
		Actions: []Action{
			{Type: ActionDeleteDB, Key: "1"},
			{Type: ActionDeleteDB, Key: "2"},
			{Type: ActionDeleteStorage, Key: "20"},
		},
	}

	opts := ReconcileOptions{
		Confirmed: true,
		DryRun:    false,
	}

	executed, err := ApplyPlan(context.Background(), spec, nil, nil, "", plan, opts)
	assert.NoError(t, err)
	assert.Equal(t, 3, executed)

	// Verify individual methods were called (fallback)
	assert.Len(t, mutator.deletedDB, 2)
	assert.Contains(t, mutator.deletedDB, "1")
	assert.Contains(t, mutator.deletedDB, "2")

	assert.Len(t, mutator.deletedStorage, 1)
	assert.Contains(t, mutator.deletedStorage, "20")
}

// TestApplyPlan_LargeScaleBatch tests performance with large number of items.
func TestApplyPlan_LargeScaleBatch(t *testing.T) {
	mutator := &mockBatchMutator{
		mockMutator: mockMutator{
			mockAdapter: mockAdapter{},
		},
		batchDBCalls:       make([][]string, 0),
		batchGamedataCalls: make([][]string, 0),
		batchStorageCalls:  make([][]string, 0),
	}

	spec := &Spec{
		Adapter:  mutator,
		CacheTTL: 0,
	}

	// Create 1000 delete actions
	actions := make([]Action, 0, 1000)
	for i := 0; i < 1000; i++ {
		actions = append(actions, Action{
			Type: ActionDeleteStorage,
			Key:  fmt.Sprintf("%d", i),
		})
	}

	plan := &ReconcilePlan{
		Actions: actions,
	}

	opts := ReconcileOptions{
		Confirmed: true,
		DryRun:    false,
	}

	executed, err := ApplyPlan(context.Background(), spec, nil, nil, "", plan, opts)
	assert.NoError(t, err)
	assert.Equal(t, 1000, executed)

	// Verify only ONE batch call was made for all 1000 items
	assert.Len(t, mutator.batchStorageCalls, 1, "Should use single batch call for 1000 items")
	assert.Len(t, mutator.batchStorageCalls[0], 1000, "Batch should contain all 1000 items")
	assert.Len(t, mutator.deletedStorage, 0, "Should NOT use individual calls")
}

// mockBatchMutator implements batch deletion methods for testing.
type mockBatchMutator struct {
	mockMutator
	batchDBCalls       [][]string
	batchGamedataCalls [][]string
	batchStorageCalls  [][]string
}

func (m *mockBatchMutator) DeleteDBBatch(ctx context.Context, keys []string) error {
	m.batchDBCalls = append(m.batchDBCalls, keys)
	return nil
}

func (m *mockBatchMutator) DeleteGamedataBatch(ctx context.Context, keys []string) error {
	m.batchGamedataCalls = append(m.batchGamedataCalls, keys)
	return nil
}

func (m *mockBatchMutator) DeleteStorageBatch(ctx context.Context, keys []string) error {
	m.batchStorageCalls = append(m.batchStorageCalls, keys)
	return nil
}
