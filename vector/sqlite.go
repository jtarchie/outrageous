package vector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"sort"
	"unsafe"

	"github.com/georgysavva/scany/v2/sqlscan"
	sqlite "github.com/mattn/go-sqlite3"
)

type Hyperplanes [][]float64

func (h *Hyperplanes) Scan(value interface{}) error {
	return json.Unmarshal([]byte(value.(string)), h)
}

type SQLite struct {
	db          *sql.DB
	hyperplanes Hyperplanes
}

var _ VectorStore = (*SQLite)(nil)

func NewSQLite(filename string, vectorDim int) (*SQLite, error) {
	db, err := sql.Open("sqlite3_vec", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS embeddings (
			id TEXT PRIMARY KEY,
			vector BLOB NOT NULL,
			content TEXT NOT NULL,
			metadata BLOB,
			hash INTEGER,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP NOT NULL
		) STRICT;

		CREATE INDEX IF NOT EXISTS idx_embeddings_hash ON embeddings (hash);

		CREATE TABLE IF NOT EXISTS hyperplanes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			config BLOB NOT NULL
		) STRICT;
	`)

	if err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	var hyperplanes Hyperplanes

	err = db.QueryRow("SELECT config FROM hyperplanes WHERE id = 1").Scan(&hyperplanes)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("hyperplanes.query", "error", err)

		return nil, fmt.Errorf("failed to query hyperplanes: %w", err)
	}

	if len(hyperplanes) == 0 {
		numHyperplanes := int(min(64, 10*math.Log2(float64(vectorDim))))

		hyperplanes = make(Hyperplanes, numHyperplanes)
		for i := range hyperplanes {
			hyperplanes[i] = make([]float64, vectorDim)
			for j := range hyperplanes[i] {
				hyperplanes[i][j] = rand.NormFloat64()
			}
		}

		content, err := json.Marshal(hyperplanes)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal hyperplanes: %w", err)
		}

		slog.Debug("hyperplanes.insert", "num_hyperplanes", numHyperplanes, "dim", vectorDim)

		_, err = db.Exec("INSERT INTO hyperplanes (config) VALUES (jsonb(?))", string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to insert hyperplanes: %w", err)
		}
	}

	slog.Debug("hyperplanes", "num_hyperplanes", len(hyperplanes), "dim", vectorDim)

	return &SQLite{
		db:          db,
		hyperplanes: hyperplanes,
	}, nil
}

func hashVector(vector Vector, hyperplanes Hyperplanes) uint64 {
	var hash uint64 = 0

	for i, plane := range hyperplanes {
		dotProduct := 0.0
		for j, val := range vector {
			dotProduct += float64(val) * plane[j]
		}
		if dotProduct > 0 {
			hash |= (1 << i) // Set the i-th bit
		}
	}

	return hash
}

func (s *SQLite) Insert(ctx context.Context, id string, vector Vector, content string, metadata map[string]string) error {
	hash := hashVector(vector, s.hyperplanes)

	metadataContents, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	vectorContents, err := json.Marshal(vector)
	if err != nil {
		return fmt.Errorf("failed to marshal vector: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		"INSERT INTO embeddings (id, vector, content, metadata, hash) VALUES (?, jsonb(?), ?, jsonb(?), ?)",
		id,
		string(vectorContents),
		content,
		string(metadataContents),
		*(*int64)(unsafe.Pointer(&hash)), // Convert uint64 to int64
	)
	if err != nil {
		slog.Error("insert.exec", "error", err)
		return fmt.Errorf("failed to insert embedding: %w", err)
	}

	return nil
}

type Metadata map[string]string

func (m *Metadata) Scan(value interface{}) error {
	return json.Unmarshal([]byte(value.(string)), m)
}

type Hash uint64

func (h *Hash) Scan(value interface{}) error {
	asInt := value.(int64)
	*h = Hash(*(*uint64)(unsafe.Pointer(&asInt))) // Convert int64 to uint64

	return nil
}

func (s *SQLite) Query(ctx context.Context, query Vector) ([]Result, error) {
	hash := hashVector(query, s.hyperplanes)

	slog.Debug("query", "hash", hash)

	var results []Result
	err := sqlscan.Select(
		ctx,
		s.db,
		&results,
		`
			SELECT
				id,
				json(vector) as vector,
				content,
				json(metadata) as metadata,
				created_at,
				hamming(:hash, hash) as distance
			FROM
				embeddings
			WHERE
				1=1
			ORDER BY
				distance ASC
			LIMIT 10
		`,
		sql.Named("hash", *(*int64)(unsafe.Pointer(&hash))), // Convert uint64 to int64
	)
	if err != nil {
		slog.Error("query.exec", "error", err)
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}

	for i := range results {
		results[i].Similarity, err = query.cosineSimilarity(results[i].Vector)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate cosine similarity: %w", err)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	return results, nil
}

func (s *SQLite) Close() error {
	err := s.db.Close()
	if err != nil {
		return fmt.Errorf("failed to close sqlite database: %w", err)
	}

	return nil
}

func hammingDistance(a, b int64) int {
	xor := *(*uint64)(unsafe.Pointer(&a)) ^ *(*uint64)(unsafe.Pointer(&b)) // XOR finds differing bits
	dist := 0

	for xor > 0 {
		dist += int(xor & 1) // Count 1s
		xor >>= 1
	}

	return dist
}

func init() {
	sql.Register("sqlite3_vec", &sqlite.SQLiteDriver{
		ConnectHook: func(conn *sqlite.SQLiteConn) error {
			if err := conn.RegisterFunc("hamming", hammingDistance, true); err != nil {
				return err
			}
			return nil
		},
	})
}
