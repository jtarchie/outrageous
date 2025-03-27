package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/examples"
	"log/slog"
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
	// examples.Run is only used to run examples
	// it allows for testing of the examples
	err := examples.Run(TriageAgentDemo)
	if err != nil {
		log.Fatal(err)
	}
}

func TriageAgentDemo() (*agent.Response, error) {
	triageAgent := agent.New(
		"Triage Agent",
		"Determine which agent is best suited to handle the user's request, and transfer the conversation to that agent.",
	)
	salesAgent := agent.New(
		"Sales Agent",
		"Be super enthusiastic about selling bees.",
	)
	refundsAgent := agent.New(
		"Refunds Agent",
		"Help the user with a refund. If the reason is that it was too expensive, offer the user a refund code. If they insist, then process the refund.",
	)

	triageAgent.Tools.Add(
		salesAgent.AsTool("Transfer the conversation to the sales agent."),
		refundsAgent.AsTool("Transfer the conversation to the refunds agent."),
	)
	salesAgent.Tools.Add(
		triageAgent.AsTool("Call this function if a user is asking about a topic that is not handled by the current agent."),
	)
	refundsAgent.Tools.Add(
		triageAgent.AsTool("Call this function if a user is asking about a topic that is not handled by the current agent."),
		agent.MustWrapStruct("Refund an item. Refund an item. Make sure you have the item_id of the form item_... Ask for user confirmation before processing the refund", ProcessRefund{}),
		agent.MustWrapStruct("Apply a discount to the user's cart", ApplyDiscount{}),
	)

	messages := agent.Messages{
		agent.Message{Role: "user", Content: "I want to buy some bees."},
	}

	response, err := triageAgent.Run(
		context.Background(),
		messages,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to run: %w", err)
	}

	return response, nil
}
