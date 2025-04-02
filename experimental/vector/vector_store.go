package vector

import "context"

type VectorStore interface {
	Insert(ctx context.Context, id string, vector Vector, content string, metadata map[string]string) error
	Query(ctx context.Context, query Vector, numResults int) ([]Result, error)
	Close() error
}
