package main

import (
	"log/slog"
	"os"

	. "github.com/jtarchie/outrageous/agent"
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

	agent := NewAgent(
		"Webpage Scraper",
		"Answers questions about webpage content by scraping and analyzing the data from the provided URLs. Please use the provided tool to scrape the webpage.",
		// WithClient(NewGeminiClient(os.Getenv("GEMINI_API_TOKEN"), "gemini-1.5-flash-8b")),
	)

	agent.Tools.Add(MustWrapStruct("This tool enables the agent to read and extract contents from an external website, providing structured data for further analysis.", tools.WebPage{}))

	messages, err := Demo(agent)
	if err != nil {
		slog.Error("execute", "error", err)

		pp.Default.SetOmitEmpty(true)
		pp.Default.SetExportedOnly(true)
		pp.Print(messages)
	}
}
