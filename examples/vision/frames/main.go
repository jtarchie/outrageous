package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/client"
	"github.com/lmittmann/tint"
	"github.com/sashabaranov/go-openai"
)

//go:embed prompt.md
var prompt string

type CLI struct {
	Video    string `help:"Path to the video file." required:""`
	Search   string `help:"Search term."`
	Limit    int    `help:"Limit the number of results." default:"10"`
	Interval int    `help:"Interval in seconds." default:"5"`
	Output   string `help:"Output directory." default:"./output" required:""`
	Model    string `help:"Model to use." default:"gemma3:12b"`
}

func (c *CLI) Run() error {
	ctx := context.Background()
	err := os.MkdirAll(c.Output, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	command := exec.CommandContext(
		ctx,
		"ffmpeg", "-i", c.Video, "-vf", fmt.Sprintf("fps=1/%d", c.Interval), fmt.Sprintf("%s/frame_%%04d.jpg", c.Output),
	)
	err = command.Run()
	if err != nil {
		return fmt.Errorf("failed to extract frames: %w", err)
	}

	frameAgent := agent.New(
		"FrameAgent",
		prompt,
		agent.WithClient(client.NewOllamaClient(c.Model)),
	)

	frames, err := doublestar.FilepathGlob(filepath.Join(c.Output, "*.jpg"))
	if err != nil {
		return fmt.Errorf("failed to find frames: %w", err)
	}

	slog.Debug("analyze.frames", "frames", len(frames))

	for index, frame := range frames {
		slog.Debug("analyze.frame", "frame", frame, "index", index)

		encodedFrame, err := agent.Base64EncodeImage(frame)
		if err != nil {
			return fmt.Errorf("failed to encode frame: %w", err)
		}

		response, err := frameAgent.Run(ctx, agent.Messages{
			agent.Message{
				Role: "user",
				MultiContent: []openai.ChatMessagePart{
					openai.ChatMessagePart{
						Type: "image_url",
						ImageURL: &openai.ChatMessageImageURL{
							URL: encodedFrame,
						},
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to get response from agent: %w", err)
		}

		slog.Debug("analyze.frame", "frame", frame, "index", index, "response", response.Messages[len(response.Messages)-1].Content)
	}

	return nil
}

func main() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: slog.LevelDebug,
	})))

	cli := &CLI{}
	ctx := kong.Parse(cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
