package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"go-be/config"
	"go-be/internal/seed"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	users := seed.DefaultUsers()
	if email := os.Getenv("SEED_ADMIN_EMAIL"); email != "" {
		users[0].Email = email
	}
	if password := os.Getenv("SEED_ADMIN_PASSWORD"); password != "" {
		users[0].Password = password
	}
	if username := os.Getenv("SEED_ADMIN_USERNAME"); username != "" {
		users[0].Username = username
	}

	log.Println("Seeding database...")
	if err := seed.Users(ctx, db, users); err != nil {
		log.Fatal("Seed failed:", err)
	}

	log.Println("Seed completed successfully")
	log.Printf("Admin login: %s / %s", users[0].Email, users[0].Password)
}
