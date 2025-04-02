package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/examples"
	"github.com/jtarchie/outrageous/tools"
)

func main() {
	// examples.Run is only used to run examples
	// it allows for testing of the examples
	fun := WebPageScraping(context.Background(), "https://eldora.com")

	err := examples.Run(fun)
	if err != nil {
		log.Fatal(err)
	}
}

func WebPageScraping(ctx context.Context, url string) func() (*agent.Response, error) {
	return func() (*agent.Response, error) {
		scrapeAgent := agent.New(
			"Webpage Scraper",
			"Answers questions about webpage content by scraping and analyzing the data from the provided URLs. Please use the provided tool to scrape the webpage.",
		)

		scrapeAgent.Tools.Add(
			agent.MustWrapStruct(
				"This tool enables the agent to read and extract contents from an external website, providing structured data for further analysis.",
				tools.WebPage{},
			),
		)

		response, err := scrapeAgent.Run(
			ctx,
			agent.Messages{
				agent.Message{Role: "user", Content: fmt.Sprintf("Please read the weather report from %s?", url)},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to run: %w", err)
		}

		return response, nil
	}
}
