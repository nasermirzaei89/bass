package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/nasermirzaei89/bass"
)

func main() {
	repo := bass.NewMemRepo()
	h := bass.NewHandler(repo)

	err := http.ListenAndServe(":8080", h) //nolint:gosec
	if err != nil {
		slog.ErrorContext(context.Background(), "error on listen and serve http", "error", err)
		os.Exit(1)
	}
}
