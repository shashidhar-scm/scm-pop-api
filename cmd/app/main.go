package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"pop/internal/db"
	"pop/internal/handlers"
	"pop/internal/middleware"
	"pop/internal/repository"
	"pop/internal/routes"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("Starting migration process...")
	if err := db.RunMigrations(ctx, "migrations"); err != nil {
		log.Printf("migration error: %v", err)
		return
	}
	log.Println("Migrations completed successfully")

	popDB, err := db.NewPopDB(context.Background())
	if err != nil {
		fmt.Println("error connecting to pop database:", err)
		return
	}
	defer popDB.Close()

	repo := repository.NewPopRepository(popDB)
	h := handlers.NewPopHandler(repo)

	mux := http.NewServeMux()
	routes.RegisterPopRoutes(mux, h)

	// Wrap entire router with rate limit then global CORS
	handler := middleware.RateLimit(mux)
	handler = middleware.CORS(handler)

	addr := ":8080"
	fmt.Println("starting API server on", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Println("server error:", err)
	}
}
