package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strings"

	. "github.com/jtarchie/outrageous/agent"
	"github.com/k0kubun/pp/v3"
	"github.com/lmittmann/tint"
)

type GetWeather struct {
	Location string `json:"location"`
}

func (g GetWeather) Call(ctx context.Context) (any, error) {
	if strings.Contains(strings.ToLower(g.Location), "nyc") {
		return `{"temp":32, "unit":"F"}`, nil
	}

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

	agent := New(
		"Agent",
		"You are a helpful agent.",
	)

	fn, err := WrapStruct("returns the current weather", GetWeather{})
	if err != nil {
		log.Fatal(err)
	}

	agent.Tools.Add(fn)

	response, err := agent.Run(
		context.Background(),
		Messages{
			Message{Role: "user", Content: "What's the weather in NYC?"},
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	pp.Default.SetOmitEmpty(true)
	pp.Default.SetExportedOnly(true)
	pp.Print(response)
}
