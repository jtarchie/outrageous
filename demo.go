package outrageous

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

func Demo(agent *Agent) (Messages, error) {
	var messages Messages
	var initLength int

	reader := bufio.NewReader(os.Stdin)

	for {
		var userInput string

		fmt.Print("\033[90mUser\033[0m: ")
		userInput, _ = reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "exit" {
			break
		}

		messages = append(messages, Message{Role: "user", Content: userInput})
		initLength = len(messages)

		response, err := agent.Run(
			context.Background(),
			messages,
		)
		if err != nil {
			slog.Error("agent.run", "error", err, "messages", messages)
			return messages, fmt.Errorf("error running agent: %w", err)
		}
		newMessages := response.Messages[initLength:]
		for _, message := range newMessages {
			if message.Role != "assistant" {
				continue
			}

			fmt.Printf("\033[94m%s\033[0m: ", message.Name)
			if message.Content != "" {
				fmt.Println(message.Content)
			}
			if len(message.ToolCalls) > 0 {
				fmt.Println("")
				for _, toolCall := range message.ToolCalls {

					fmt.Printf("\033[95m%s\033[0m(%s)\n", toolCall.Function.Name, strings.ReplaceAll(toolCall.Function.Arguments, ":", "="))
				}
			}
		}

		messages = append(messages, newMessages...)
		agent = response.Agent
	}

	return messages, nil
}
