package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"gostock/config"
	"gostock/internal/pkg/database"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ Warning: .env file not found or failed to read. Loading configs from system environment only: %v", err)
	}

	cfg := config.LoadConfig()

	var migrationsDir string
	flag.StringVar(&migrationsDir, "dir", "./sql", "directory with migration files")
	flag.Parse()

	// Connect to the database
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("goose: failed to connect to DB: %v\n", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("goose: failed to close DB: %v\n", err)
		}
	}()

	goose.SetLogger(goose.NopLogger()) // Suppress verbose logging from goose if needed, or use a custom logger

	arguments := flag.Args()
	if len(arguments) == 0 {
		arguments = []string{"up"} // Default to 'up' if no command is provided
	}

	command := arguments[0]
	var args []string
	if len(arguments) > 1 {
		args = arguments[1:]
	}

	if err := goose.Run(command, db, migrationsDir, args...); err != nil {
		log.Fatalf("goose %v: %v", command, err)
	}

	fmt.Printf("goose %s success\n", command)
}
