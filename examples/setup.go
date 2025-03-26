package examples

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jtarchie/outrageous/agent"
	"github.com/k0kubun/pp/v3"
	"github.com/lmittmann/tint"
)

// this was extracted from examples to support testing
func Run(setup func() (*agent.Response, error)) error {
	// allow different log levels
	var logLevel slog.Level
	err := logLevel.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))
	if err != nil {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: logLevel,
	})))

	// defaults for pretty print
	pp.Default.SetOmitEmpty(true)
	pp.Default.SetExportedOnly(true)

	response, err := setup()
	if err != nil {
		slog.Error("failed to setup", "error", err)
		return fmt.Errorf("failed to setup: %w", err)
	}

	_, err = pp.Print(response)
	if err != nil {
		slog.Error("failed to print", "error", err)
		return fmt.Errorf("failed to print: %w", err)
	}

	return nil
}
