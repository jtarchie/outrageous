package main

import (
	"context"
	"log/slog"
	"os"

	. "github.com/jtarchie/outrageous"
	"github.com/k0kubun/pp/v3"
	"github.com/lmittmann/tint"
)

type ProcessRefund struct {
	ItemID string `json:"item_id"`
	Reason string `json:"reason,omitempty"`
}

func (p ProcessRefund) Call(ctx context.Context) (any, error) {
	slog.Info("process_refund", "item_id", p.ItemID, "reason", p.Reason)
	return "Success", nil
}

type ApplyDiscount struct{}

func (a ApplyDiscount) Call(ctx context.Context) (any, error) {
	slog.Info("apply_discount")
	return "Applied discount of 11%", nil
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

	triageAgent := NewAgent(
		"Triage Agent",
		"Determine which agent is best suited to handle the user's request, and transfer the conversation to that agent.",
	)
	salesAgent := NewAgent(
		"Sales Agent",
		"Be super enthusiastic about selling bees.",
	)
	refundsAgent := NewAgent(
		"Refunds Agent",
		"Help the user with a refund. If the reason is that it was too expensive, offer the user a refund code. If they insist, then process the refund.",
	)

	triageAgent.Functions.Add(
		salesAgent.AsFunction("Transfer the conversation to the sales agent."),
		refundsAgent.AsFunction("Transfer the conversation to the refunds agent."),
	)
	salesAgent.Functions.Add(
		triageAgent.AsFunction("Call this function if a user is asking about a topic that is not handled by the current agent."),
	)
	refundsAgent.Functions.Add(
		triageAgent.AsFunction("Call this function if a user is asking about a topic that is not handled by the current agent."),
		MustWrapStruct("Refund an item. Refund an item. Make sure you have the item_id of the form item_... Ask for user confirmation before processing the refund", ProcessRefund{}),
		MustWrapStruct("Apply a discount to the user's cart", ApplyDiscount{}),
	)

	messages, err := Demo(triageAgent)
	if err != nil {
		slog.Error("execute", "error", err)

		pp.Default.SetOmitEmpty(true)
		pp.Default.SetExportedOnly(true)
		pp.Print(messages)
	}
}
