package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	. "github.com/jtarchie/outrageous/agent" //nolint: staticcheck
	"github.com/jtarchie/outrageous/tools"
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
		"Webpage Scraper",
		"Answers questions about webpage content by scraping and analyzing the data from the provided URLs. Please use the provided tool to scrape the webpage.",
		// WithClient(NewGeminiClient(os.Getenv("GEMINI_API_TOKEN"), "gemini-1.5-flash-8b")),
	)

	// this uses the web page scraper tool provided by the outrageous library
	agent.Tools.Add(
		MustWrapStruct(
			"This tool enables the agent to read and extract contents from an external website, providing structured data for further analysis.",
			tools.WebPage{},
		),
	)

	response, err := agent.Run(
		context.Background(),
		Messages{
			Message{Role: "user", Content: "Please read the weather report from https://eldora.com?"},
		},
	)
	if err != nil {
		slog.Error("execute", "error", err)
	}

	pp.Default.SetOmitEmpty(true)
	pp.Default.SetExportedOnly(true)
	_, err = pp.Print(response)
	if err != nil {
		log.Fatal(err)
	}
}
