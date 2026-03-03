package db

import (
	"context"
	"database/sql"
	"fmt"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/ncruces"
	_ "github.com/asg017/sqlite-vec-go-bindings/ncruces" // replaces go-sqlite3/embed
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

// Store wraps database operations with vector search capabilities
type Store struct {
	conn    *sql.DB
	queries *Queries
}

func Open(path string) (*Store, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return &Store{
		conn:    conn,
		queries: New(conn),
	}, nil
}

// OpenMem opens a database from bytes in memory using the memdb VFS
func OpenMem(name string, data []byte) (*Store, error) {
	memdb.Create(name, data)
	conn, err := sql.Open("sqlite3", "file:/"+name+"?vfs=memdb")
	if err != nil {
		return nil, fmt.Errorf("open memdb: %w", err)
	}
	return &Store{
		conn:    conn,
		queries: New(conn),
	}, nil
}

func (s *Store) Close() error {
	return s.conn.Close()
}

func (s *Store) Init() error {
	schema := `
		CREATE TABLE IF NOT EXISTS signals (
			id INTEGER PRIMARY KEY,
			path TEXT UNIQUE NOT NULL,
			description TEXT NOT NULL,
			type TEXT NOT NULL,
			datatype TEXT NOT NULL,
			unit TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_signals_path ON signals(path);
		CREATE INDEX IF NOT EXISTS idx_signals_type ON signals(type);
		CREATE VIRTUAL TABLE IF NOT EXISTS vec_signals USING vec0(
			id INTEGER PRIMARY KEY,
			embedding FLOAT[1536]
		);
	`
	_, err := s.conn.Exec(schema)
	return err
}

func (s *Store) InsertSignal(ctx context.Context, path, description, typ, datatype, unit string, embedding []float64) error {
	// Insert metadata
	id, err := s.queries.InsertSignal(ctx, InsertSignalParams{
		Path:        path,
		Description: description,
		Type:        typ,
		Datatype:    datatype,
		Unit:        unit,
	})
	if err != nil {
		return fmt.Errorf("insert signal metadata: %w", err)
	}

	// Convert float64 to float32 for sqlite-vec
	emb32 := make([]float32, len(embedding))
	for i, v := range embedding {
		emb32[i] = float32(v)
	}

	// Serialize and insert embedding
	vec, err := sqlite_vec.SerializeFloat32(emb32)
	if err != nil {
		return fmt.Errorf("serialize embedding: %w", err)
	}

	_, err = s.conn.ExecContext(ctx, "INSERT INTO vec_signals(id, embedding) VALUES (?, ?)", id, vec)
	if err != nil {
		return fmt.Errorf("insert embedding: %w", err)
	}

	return nil
}

type SearchResult struct {
	Signal   Signal
	Distance float64
}

func (s *Store) Search(ctx context.Context, queryEmbedding []float64, limit int) ([]SearchResult, error) {
	// Convert float64 to float32
	emb32 := make([]float32, len(queryEmbedding))
	for i, v := range queryEmbedding {
		emb32[i] = float32(v)
	}

	vec, err := sqlite_vec.SerializeFloat32(emb32)
	if err != nil {
		return nil, fmt.Errorf("serialize query embedding: %w", err)
	}

	// KNN search using vec0
	rows, err := s.conn.QueryContext(ctx, `
		SELECT
			v.id,
			v.distance,
			s.path,
			s.description,
			s.type,
			s.datatype,
			s.unit
		FROM vec_signals v
		JOIN signals s ON s.id = v.id
		WHERE v.embedding MATCH ? AND k = ?
		ORDER BY v.distance
	`, vec, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var id int64
		if err := rows.Scan(&id, &r.Distance, &r.Signal.Path, &r.Signal.Description, &r.Signal.Type, &r.Signal.Datatype, &r.Signal.Unit); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		r.Signal.ID = id
		results = append(results, r)
	}

	return results, rows.Err()
}
