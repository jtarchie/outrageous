package main

import (
	"log/slog"
	"os"

	. "github.com/jtarchie/outrageous"
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
		"Answers questions about webpage content.",
	)

	agent.Tools.Add(MustWrapStruct("read contents of a webpage", tools.WebPage{}))


	messages, err := Demo(agent)
	if err != nil {
		slog.Error("execute", "error", err)

		pp.Default.SetOmitEmpty(true)
		pp.Default.SetExportedOnly(true)
		pp.Print(messages)
	}
}
