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

	if err := db.RunMigrations(databaseURL); err != nil {
		log.Fatal("Migration failed: ", err)
	}
	log.Println("Migrations applied successfully")
}
