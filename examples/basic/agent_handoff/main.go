package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	. "github.com/jtarchie/outrageous"
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

	englishAgent := NewAgent(
		"English Agent",
		"You only speak English",
	)

	spanishAgent := NewAgent(
		"Spanish Agent",
		"You only speak Spanish",
	)

	englishAgent.Functions.Add(spanishAgent.AsFunction("Transfer spanish speaking users immediately"))

	response, err := englishAgent.Run(
		context.Background(),
		Messages{
			Message{Role: "user", Content: "Hola. ¿Como estás?"},
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	pp.Default.SetOmitEmpty(true)
	pp.Default.SetExportedOnly(true)
	pp.Print(response)
}
