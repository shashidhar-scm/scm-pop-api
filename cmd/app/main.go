package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"pop/internal/db"
	"pop/internal/handlers"
	"pop/internal/repository"
	"pop/internal/routes"
	"pop/internal/middleware"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := db.RunMigrations(ctx, "migrations"); err != nil {
		fmt.Println("migration error:", err)
		return
	}

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

	// Wrap entire router with global CORS
	handler := middleware.CORS(mux)

	addr := ":8080"
	fmt.Println("starting API server on", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Println("server error:", err)
	}
}
