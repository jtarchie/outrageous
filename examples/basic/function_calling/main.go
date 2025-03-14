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

type GetWeather struct {
	Location string `json:"location"`
}

func (g GetWeather) Call(ctx context.Context) (any, error) {
	return `{"temp":67, "unit":"F"}`, nil
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

	agent := NewAgent(
		"Agent",
		"You are a helpful agent.",
	)

	fn, err := WrapStruct("returns the current weather", GetWeather{})
	if err != nil {
		log.Fatal(err)
	}

	agent.Functions.Add(fn)

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
	pp.Print(response)
}
