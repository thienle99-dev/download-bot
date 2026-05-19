package main

import (
	"context"
	"download-bot/internal/bot"
	"download-bot/internal/config"
	"download-bot/internal/storage"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Initializing Download Bot server...")

	// 1. Load config
	cfg := config.Load()
	if cfg.BotToken == "" {
		log.Fatal("CRITICAL: BOT_TOKEN is required to start the bot. Please set it in your environment.")
	}

	// 2. Initialize Database
	db, err := storage.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("CRITICAL: Failed to initialize SQLite database: %v", err)
	}
	defer db.Close()

	// 3. Create Bot Server
	server, err := bot.NewBotServer(cfg, db)
	if err != nil {
		log.Fatalf("CRITICAL: Failed to create bot server: %v", err)
	}

	// 3.5. Start HTTP server to serve downloaded files and admin dashboard
	go func() {
		mux := http.NewServeMux()
		server.RegisterWebRoutes(mux)
		log.Printf("Starting HTTP web server on %s", cfg.ServerPort)
		if err := http.ListenAndServe(cfg.ServerPort, mux); err != nil {
			log.Printf("Error running HTTP web server: %v", err)
		}
	}()

	// Create a context that is cancelled when we receive an OS signal
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Handle graceful shutdown logging
	go func() {
		<-ctx.Done()
		log.Println("Received termination signal. Shutting down gracefully...")
	}()

	// 4. Start the Bot listener (blocks until context is cancelled)
	server.Start(ctx)
	log.Println("Bot server stopped.")
}
