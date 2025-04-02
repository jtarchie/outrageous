package tools

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
)

// tool that will scrape a webpage and return the title, description, and body
type WebPage struct {
	Url string `json:"url"`
}

// the page response, which will be returned to the agent
type WebPageResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

func (w WebPage) Call(ctx context.Context) (any, error) {
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	response := WebPageResponse{}
	err := chromedp.Run(ctx,
		chromedp.Navigate(w.Url),
		chromedp.Text("title", &response.Title),
		chromedp.Text("meta[name='description']", &response.Description),
		chromedp.Text("body", &response.Body),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scrape webpage: %w", err)
	}

	return response, nil
}
