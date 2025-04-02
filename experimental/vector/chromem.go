package vector

import (
	"context"
	"fmt"

	"github.com/philippgille/chromem-go"
)

type Chromem struct {
	collection *chromem.Collection
}

var _ VectorStore = (*Chromem)(nil)

func NewChromem() (*Chromem, error) {
	db := chromem.NewDB()

	collection, err := db.CreateCollection("documents", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return &Chromem{
		collection: collection,
	}, nil
}

func (c *Chromem) Close() error {
	return nil
}

func (c *Chromem) Insert(ctx context.Context, id string, vector Vector, content string, metadata map[string]string) error {
	err := c.collection.AddDocument(ctx, chromem.Document{
		Content:   content,
		Embedding: vector,
		ID:        id,
		Metadata:  metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	return nil
}

func (c *Chromem) Query(ctx context.Context, query Vector, numResults int) ([]Result, error) {
	results, err := c.collection.QueryEmbedding(ctx, query, numResults, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	var res []Result
	for _, r := range results {
		res = append(res, Result{
			Content:  r.Content,
			Vector:   r.Embedding,
			ID:       r.ID,
			Metadata: r.Metadata,

			Similarity: r.Similarity,
		})
	}
	return res, nil
}
