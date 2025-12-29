package main

import (
	"context"
	"fmt"
	"time"

	"pop/internal/db"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := db.RunMigrations(ctx, "migrations"); err != nil {
		fmt.Println("migration error:", err)
		return
	}

	fmt.Println("migrations applied successfully")
}
