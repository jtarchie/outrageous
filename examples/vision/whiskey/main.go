package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/client"
	"github.com/lmittmann/tint"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed prompt.md
var prompt string

type CLI struct {
	Filenames  []string `arg:"" type:"existingfile" help:"Image file to rename"`
	ImageModel string   `help:"OpenAI image model" default:"gemma3:latest" required:""`
	Database   string   `help:"SQLite database file" default:"whiskey.db"`
}

func (c *CLI) Run() error {
	db, err := setupDatabase(c.Database)
	if err != nil {
		return fmt.Errorf("failed to set up database: %w", err)
	}
	defer func() { _ = db.Close() }()

	bottleAgent := agent.New(
		"WhiskeyAgent",
		prompt,
		agent.WithClient(client.NewOllamaClient(c.ImageModel)),
		// agent.WithClient(client.NewOpenAIClient(os.Getenv("OPENAI_API_KEY"), c.ImageModel)),
		// agent.WithClient(client.NewGeminiClient(os.Getenv("GEMINI_API_TOKEN"), c.ImageModel)),
		// agent.WithClient(client.NewAnthropicClient(os.Getenv("ANTHROPIC_API_TOKEN"), c.ImageModel)),
	)

	err = bottleAgent.SetSchema(BottleSchema{})
	if err != nil {
		return fmt.Errorf("failed to set schema: %w", err)
	}

	for _, filename := range c.Filenames {
		response, err := bottleAgent.Run(context.Background(), agent.Messages{
			agent.ImageMessage(filename),
		})
		if err != nil {
			return fmt.Errorf("failed to run agent: %w", err)
		}

		payload := response.Messages[len(response.Messages)-1].Content

		slog.Debug("agent.run", "filename", filename, "payload", payload)

		_, err = db.Exec(
			"INSERT INTO bottles (payload, created_at, filename, model) VALUES (?, ?, ?, ?)",
			payload, time.Now(), filename, c.ImageModel,
		)
		if err != nil {
			return fmt.Errorf("failed to insert bottle into database: %w", err)
		}
	}

	return nil
}

func setupDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create bottles table if it doesn't exist
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS bottles (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            payload TEXT NOT NULL,
						filename TEXT NOT NULL,
            created_at TEXT NOT NULL,
						model TEXT NOT NULL
        ) STRICT;
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to create bottles table: %w", err)
	}

	return db, nil
}

func main() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: slog.LevelDebug,
	})))

	cli := &CLI{}
	ctx := kong.Parse(cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
