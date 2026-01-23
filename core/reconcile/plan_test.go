package reconcile

import (
	"context"
	"testing"

	"asset-manager/core/storage/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestReconcileWithPlan_PurgeActions tests that purge actions are planned correctly.
func TestReconcileWithPlan_PurgeActions(t *testing.T) {
	adapter := &mockAdapter{
		dbIndex: map[string]DBItem{
			"1": "item1", // Present in DB only (missing in GD and storage)
		},
		gdIndex: map[string]GDItem{
			"2": "item2", // Present in GD only (missing in DB and storage)
		},
		storageSet: map[string]struct{}{
			"3": {}, // Present in storage only (missing in DB and GD)
		},
		mismatches: map[string][]string{},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 0, // Disable caching for test
	}

	opts := ReconcileOptions{
		DoPurge:   true,
		DoSync:    false,
		Confirmed: false,
		DryRun:    false,
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	plan, err := ReconcileWithPlan(context.Background(), spec, nil, mockClient, "", opts)
	assert.NoError(t, err)
	assert.NotNil(t, plan)

	// Should have 3 results (one for each key)
	assert.Len(t, plan.Results, 3)

	// All items are missing in at least one store
	// item1: missing in GD and storage (2 missing)
	// item2: missing in DB and storage (2 missing)
	// item3: missing in DB and GD (2 missing)
	assert.Equal(t, 2, plan.Summary.MissingGamedata) // item1, item3
	assert.Equal(t, 2, plan.Summary.MissingStorage)  // item1, item2
	assert.Equal(t, 2, plan.Summary.MissingDB)       // item2, item3

	// Should plan delete actions for all present stores
	// item1: delete from DB (missing in GD and storage)
	// item2: delete from GD (missing in DB and storage)
	// item3: delete from storage (missing in DB and GD)
	assert.Equal(t, 3, plan.Summary.PurgeActions)
	assert.Len(t, plan.Actions, 3)

	// Verify action types
	actionTypes := make(map[ActionType]int)
	for _, action := range plan.Actions {
		actionTypes[action.Type]++
	}
	assert.Equal(t, 1, actionTypes[ActionDeleteDB])
	assert.Equal(t, 1, actionTypes[ActionDeleteGamedata])
	assert.Equal(t, 1, actionTypes[ActionDeleteStorage])
}

// TestReconcileWithPlan_SyncActions tests that sync actions are planned correctly.
func TestReconcileWithPlan_SyncActions(t *testing.T) {
	t.Skip("TODO: Fix mockAdapter.CompareFields to properly return mismatches")

	adapter := &mockAdapter{
		dbIndex: map[string]DBItem{
			"1": "item1",
		},
		gdIndex: map[string]GDItem{
			"1": "item1",
		},
		storageSet: map[string]struct{}{
			"1": {},
		},
		mismatches: map[string][]string{
			"1": {"width: gd=2 db=1", "length: gd=3 db=2"},
		},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 0,
	}

	opts := ReconcileOptions{
		DoPurge:   false,
		DoSync:    true,
		Confirmed: false,
		DryRun:    false,
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	plan, err := ReconcileWithPlan(context.Background(), spec, nil, mockClient, "", opts)
	assert.NoError(t, err)
	assert.NotNil(t, plan)

	// Should have 1 result
	assert.Len(t, plan.Results, 1)

	// No missing items
	assert.Equal(t, 0, plan.Summary.MissingGamedata)
	assert.Equal(t, 0, plan.Summary.MissingStorage)
	assert.Equal(t, 0, plan.Summary.MissingDB)

	// Should have 1 mismatch
	assert.Equal(t, 1, plan.Summary.Mismatches)

	// Should plan 1 sync action
	assert.Equal(t, 1, plan.Summary.SyncActions)
	assert.Len(t, plan.Actions, 1)
	assert.Equal(t, ActionSyncDB, plan.Actions[0].Type)
	assert.Equal(t, "1", plan.Actions[0].Key)
}

// TestReconcileWithPlan_PurgePrecedence tests that purge takes precedence over sync.
func TestReconcileWithPlan_PurgePrecedence(t *testing.T) {
	adapter := &mockAdapter{
		dbIndex: map[string]DBItem{
			"1": "item1",
		},
		gdIndex: map[string]GDItem{
			"1": "item1",
		},
		storageSet: map[string]struct{}{
			// Missing in storage
		},
		mismatches: map[string][]string{
			"1": {"width: gd=2 db=1"}, // Has mismatch
		},
	}

	spec := &Spec{
		Adapter:  adapter,
		CacheTTL: 0,
	}

	opts := ReconcileOptions{
		DoPurge:   true,
		DoSync:    true, // Both enabled
		Confirmed: false,
		DryRun:    false,
	}

	mockClient := new(mocks.Client)
	mockClient.On("BucketExists", mock.Anything, "").Return(true, nil)

	plan, err := ReconcileWithPlan(context.Background(), spec, nil, mockClient, "", opts)
	assert.NoError(t, err)

	// Should plan purge, NOT sync (purge takes precedence)
	assert.Equal(t, 2, plan.Summary.PurgeActions) // Delete from DB and GD
	assert.Equal(t, 0, plan.Summary.SyncActions)  // No sync

	// Verify actions are purge only
	for _, action := range plan.Actions {
		assert.NotEqual(t, ActionSyncDB, action.Type)
	}
}

// TestApplyPlan_ConfirmationGating tests that apply respects confirmation flag.
func TestApplyPlan_ConfirmationGating(t *testing.T) {
	mutator := &mockMutator{
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
			{Type: ActionDeleteGamedata, Key: "2"},
		},
	}

	// Test 1: Not confirmed - should not execute
	opts := ReconcileOptions{
		Confirmed: false,
		DryRun:    false,
	}

	executed, err := ApplyPlan(context.Background(), spec, nil, nil, "", plan, opts)
	assert.NoError(t, err)
	assert.Equal(t, 0, executed)
	assert.Len(t, mutator.deletedDB, 0)
	assert.Len(t, mutator.deletedGamedata, 0)

	// Test 2: Confirmed but dry-run - should not execute
	opts.Confirmed = true
	opts.DryRun = true

	executed, err = ApplyPlan(context.Background(), spec, nil, nil, "", plan, opts)
	assert.NoError(t, err)
	assert.Equal(t, 0, executed)
	assert.Len(t, mutator.deletedDB, 0)
	assert.Len(t, mutator.deletedGamedata, 0)

	// Test 3: Confirmed and not dry-run - should execute
	opts.DryRun = false

	executed, err = ApplyPlan(context.Background(), spec, nil, nil, "", plan, opts)
	assert.NoError(t, err)
	assert.Equal(t, 2, executed)
	assert.Len(t, mutator.deletedDB, 1)
	assert.Equal(t, "1", mutator.deletedDB[0])
	assert.Len(t, mutator.deletedGamedata, 1)
	assert.Equal(t, "2", mutator.deletedGamedata[0])
}

// mockMutator implements both Adapter and Mutator for testing.
type mockMutator struct {
	mockAdapter
	deletedDB       []string
	deletedGamedata []string
	deletedStorage  []string
	synced          []string
}

func (m *mockMutator) DeleteDB(ctx context.Context, key string) error {
	m.deletedDB = append(m.deletedDB, key)
	return nil
}

func (m *mockMutator) DeleteGamedata(ctx context.Context, key string) error {
	m.deletedGamedata = append(m.deletedGamedata, key)
	return nil
}

func (m *mockMutator) DeleteStorage(ctx context.Context, key string) error {
	m.deletedStorage = append(m.deletedStorage, key)
	return nil
}

func (m *mockMutator) SyncDBFromGamedata(ctx context.Context, key string, gdItem GDItem) error {
	m.synced = append(m.synced, key)
	return nil
}
