package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/lmittmann/tint"
	"github.com/onsi/gomega"
)

func TestScraping(t *testing.T) {
	t.SkipNow()

	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: slog.LevelDebug,
	})))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>Weather Update</title>
			</head>
			<body>
				<h1>Fresh powder on the slopes today!</h1>
				<p>Temperature: 28F</p>
				<p>Conditions: Sunny</p>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	a := gomega.NewGomegaWithT(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fun := WebPageScraping(ctx, server.URL)
	response, err := fun()
	a.Expect(err).NotTo(gomega.HaveOccurred())

	a.Expect(response.Messages).To(gomega.HaveLen(5))
	a.Expect(response.Agent.Name()).To(gomega.Equal("WebpageScraper"))
}
