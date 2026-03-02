package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	_, statErr := os.Stat(dbPath)
	if statErr != nil && !os.IsNotExist(statErr) {
		t.Fatalf("unexpected stat error: %v", statErr)
	}
}

func TestInit(t *testing.T) {
	store, skip := setupTestStore(t)
	if skip {
		return
	}
	defer store.Close()

	ctx := context.Background()

	var count int64
	err := store.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM signals").Scan(&count)
	if err != nil {
		t.Fatalf("signals table not created: %v", err)
	}

	_, err = store.conn.ExecContext(ctx, "SELECT * FROM vec_signals LIMIT 1")
	if err != nil {
		t.Fatalf("vec_signals virtual table not created: %v", err)
	}
}

func TestInsertSignal(t *testing.T) {
	store, skip := setupTestStore(t)
	if skip {
		return
	}
	defer store.Close()

	ctx := context.Background()

	embedding := make([]float64, 1536)
	for i := range embedding {
		embedding[i] = float64(i) / 1536.0
	}

	err := store.InsertSignal(ctx, "Vehicle.Speed", "Vehicle speed", "sensor", "float", "km/h", embedding)
	if err != nil {
		t.Fatalf("InsertSignal failed: %v", err)
	}

	signal, err := store.queries.GetSignalByPath(ctx, "Vehicle.Speed")
	if err != nil {
		t.Fatalf("GetSignalByPath failed: %v", err)
	}

	if signal.Path != "Vehicle.Speed" {
		t.Errorf("expected path 'Vehicle.Speed', got '%s'", signal.Path)
	}
	if signal.Description != "Vehicle speed" {
		t.Errorf("expected description 'Vehicle speed', got '%s'", signal.Description)
	}
}

func TestSearch(t *testing.T) {
	store, skip := setupTestStore(t)
	if skip {
		return
	}
	defer store.Close()

	ctx := context.Background()

	signals := []struct {
		path        string
		description string
		embedding   []float64
	}{
		{"Vehicle.Speed", "Current vehicle speed", normalizedEmbedding(0.1, 0.2, 0.3)},
		{"Vehicle.Powertrain.CombustionEngine.Speed", "Engine RPM", normalizedEmbedding(0.1, 0.2, 0.31)},
		{"Vehicle.Cabin.HVAC.Temperature", "Cabin temperature", normalizedEmbedding(0.9, 0.1, 0.1)},
	}

	for _, s := range signals {
		err := store.InsertSignal(ctx, s.path, s.description, "sensor", "float", "", s.embedding)
		if err != nil {
			t.Fatalf("InsertSignal failed for %s: %v", s.path, err)
		}
	}

	queryEmb := normalizedEmbedding(0.1, 0.2, 0.3)
	results, err := store.Search(ctx, queryEmb, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Signal.Path != "Vehicle.Speed" {
		t.Errorf("expected first result 'Vehicle.Speed', got '%s'", results[0].Signal.Path)
	}

	if results[0].Distance > 0.001 {
		t.Errorf("expected distance ~0 for exact match, got %f", results[0].Distance)
	}
}

func setupTestStore(t *testing.T) (*Store, bool) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := store.Init(); err != nil {
		store.Close()
		if isWASMFeatureError(err) {
			t.Skip("Skipping: sqlite-vec WASM features not available")
			return nil, true
		}
		t.Fatalf("Init failed: %v", err)
	}

	return store, false
}

func isWASMFeatureError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "atomic") || contains(errStr, "feature") || contains(errStr, "disabled")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func normalizedEmbedding(a, b, c float64) []float64 {
	emb := make([]float64, 1536)
	emb[0] = a
	emb[1] = b
	emb[2] = c

	var sum float64
	for _, v := range emb {
		sum += v * v
	}
	if sum > 0 {
		norm := 1.0 / sqrt(sum)
		for i := range emb {
			emb[i] *= norm
		}
	}
	return emb
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
