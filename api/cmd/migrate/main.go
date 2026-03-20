package main

import (
	"log"
	"os"

	"github.com/WahyuS002/uploy/db"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}

	switch direction {
	case "up":
		if err := db.RunMigrations(databaseURL); err != nil {
			log.Fatal("Migration failed: ", err)
		}
		log.Println("Migrations applied successfully")
	case "down":
		if err := db.RollbackMigration(databaseURL); err != nil {
			log.Fatal("Rollback failed: ", err)
		}
		log.Println("Rolled back one migration")
	default:
		log.Fatalf("Unknown command: %s (use 'up' or 'down')", direction)
	}
}
