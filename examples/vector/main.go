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
)

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

func execute() error {
	model, err := fastembed.NewFlagEmbedding(&fastembed.InitOptions{
		Model:    fastembed.BGEBaseENV15,
		CacheDir: "../../.onnx_local_cache",
	})
	if err != nil {
		return fmt.Errorf("failed to create embedding model: %w", err)
	}

	db, err := vector.NewSQLite(":memory:", 768)
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}
	defer db.Close()

	file, err := os.Open("hamlet.txt")
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	chunks := [][]string{[]string{}}
	maxChunkSize := 256

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 0 {
			chunks[len(chunks)-1] = append(chunks[len(chunks)-1], "passage: "+line)
		}
		if len(chunks[len(chunks)-1]) == maxChunkSize {
			chunks = append(chunks, []string{})
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	slog.Info("embedding", "chunks", len(chunks), "max_chunk_size", maxChunkSize)

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

	query, err := model.QueryEmbed("that is the question")
	if err != nil {
		return fmt.Errorf("failed to embed query: %w", err)
	}

	slog.Info("query", "embedding", len(query))

	results, err := db.Query(context.Background(), query)
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
