package main

import (
	"fmt"
	"github.com/nasermirzaei89/bass"
	"net/http"
)

func main() {
	repo := bass.NewMemRepo()
	h := bass.NewHandler(repo)

	err := http.ListenAndServe(":8080", h) //nolint:gosec
	panic(fmt.Errorf("error on listen and serve http: %w", err))
}
