package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	. "github.com/jtarchie/outrageous/agent"
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

	agent := New(
		"Agent",
		"You are a helpful agent.",
	)

	response, err := agent.Run(
		context.Background(),
		Messages{
			Message{Role: "user", Content: "Hi"},
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	pp.Default.SetOmitEmpty(true)
	pp.Default.SetExportedOnly(true)
	pp.Print(response)
}
