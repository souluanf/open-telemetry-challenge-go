package main

import (
	"context"
	"log"
	"net/http"
	"open-telemetry-challenge-go/api"
	"open-telemetry-challenge-go/internal/tracing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	tp := tracing.InitTracer()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Get("/{cep}", api.HandleRequest)

	if err := http.ListenAndServe(":8081", router); err != nil {
		log.Fatal(err)
	}
}
