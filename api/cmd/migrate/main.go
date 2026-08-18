package main

import (
	"github.com/WahyuS002/uploy/telemetry"
	"os"

	"github.com/WahyuS002/uploy/db"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		telemetry.Fatal("DATABASE_URL is required")
	}

	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}

	switch direction {
	case "up":
		if err := db.RunMigrations(databaseURL); err != nil {
			telemetry.Fatal("Migration failed: ", err)
		}
		telemetry.Println("Migrations applied successfully")
	case "down":
		if err := db.RollbackMigration(databaseURL); err != nil {
			telemetry.Fatal("Rollback failed: ", err)
		}
		telemetry.Println("Rolled back one migration")
	default:
		telemetry.Fatalf("Unknown command: %s (use 'up' or 'down')", direction)
	}
}
