package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	// "strconv"
	"strings"

	"github.com/anush008/fastembed-go"
	"github.com/jtarchie/outrageous/vector"
	"github.com/k0kubun/pp/v3"
	"github.com/lmittmann/tint"
	"github.com/samber/lo"
)

func Cons[T any](elements []T, size int) [][]T {
	var results [][]T
	for index := range elements {
		if index+size <= len(elements) {
			results = append(results, elements[index:index+size])
		} else {
			break
		}
	}

	return results
}

func main() {
	var logLevel slog.Level
	err := logLevel.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))
	if err != nil {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: logLevel,
	})))

	err = execute()
	if err != nil {
		slog.Error("executing", "err", err)
	}
}

var (
	model = fastembed.BGEBaseENV15
	dim   = 768
)

func execute() error {
	model, err := fastembed.NewFlagEmbedding(&fastembed.InitOptions{
		Model:    model,
		CacheDir: "../../.onnx_local_cache",
	})
	if err != nil {
		return fmt.Errorf("failed to create embedding model: %w", err)
	}

	file, err := os.Open("hamlet.txt")
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	lines := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	maxChunkSize := 256
	slog.Info("embedding", "lines", len(lines))
	chunks := lo.Chunk(
		lo.Map(
			Cons(lines, 2),
			func(element []string, _ int) string {
				return strings.Join(element, " ")
			},
		),
		maxChunkSize)

	db, err := vector.NewSQLite(":memory:", dim, len(chunks)*maxChunkSize)
	// db, err := vector.NewChromem()
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}
	defer db.Close()

	for chunkIndex, chunk := range chunks {
		slog.Info("chunk", "index", chunkIndex, "size", len(chunk))

		embeddings, err := model.Embed(chunk, len(chunk))
		if err != nil {
			return fmt.Errorf("failed to embed chunk: %w", err)
		}

		for embeddingIndex, embedding := range embeddings {
			id := strconv.Itoa(maxChunkSize*(chunkIndex+1) + embeddingIndex)
			err = db.Insert(context.Background(),
				id, embedding, chunk[embeddingIndex],
				map[string]string{
					"id": id,
				},
			)
			if err != nil {
				return fmt.Errorf("failed to insert embedding: %w", err)
			}
		}
	}

	query, err := model.Embed([]string{"to be or not to be"}, 0)
	if err != nil {
		return fmt.Errorf("failed to embed query: %w", err)
	}

	slog.Info("query", "embedding", len(query))

	results, err := db.Query(context.Background(), query[0])
	if err != nil {
		return fmt.Errorf("failed to query database: %w", err)
	}

	pp.Default.SetOmitEmpty(true)
	pp.Default.SetExportedOnly(true)

	for _, result := range results {
		slog.Info("result", "id", result.ID, "content", result.Content, "metadata", result.Metadata, "distance", result.Distance, "similarity", result.Similarity)
	}

	return nil
}
