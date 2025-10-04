package main

import (
	"context"
	"database/sql"
	"fmt"
	"librecash/config"
	librecashContext "librecash/context"
	"librecash/menu"
	"librecash/metrics"
	"librecash/rabbit"
	"librecash/repository"
	"librecash/sender"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq" // PostgreSQL driver
)

const PID_FILE = "librecash_bot.pid"

// createPidFile creates a PID file and locks it to prevent multiple instances
func createPidFile() error {
	// Check if PID file already exists
	if _, err := os.Stat(PID_FILE); err == nil {
		// PID file exists, check if process is still running
		pidBytes, err := os.ReadFile(PID_FILE)
		if err == nil {
			if pid, err := strconv.Atoi(string(pidBytes)); err == nil {
				// Check if process with this PID is still running
				if process, err := os.FindProcess(pid); err == nil {
					// Try to send signal 0 to check if process exists
					if err := process.Signal(syscall.Signal(0)); err == nil {
						return fmt.Errorf("LibreCash bot is already running with PID %d. Stop the existing instance first.", pid)
					}
				}
			}
		}
		// If we reach here, the PID file exists but process is not running
		log.Printf("[MAIN] Found stale PID file, removing it")
		os.Remove(PID_FILE)
	}

	// Create new PID file
	currentPid := os.Getpid()
	pidContent := fmt.Sprintf("%d", currentPid)

	if err := os.WriteFile(PID_FILE, []byte(pidContent), 0644); err != nil {
		return fmt.Errorf("failed to create PID file: %v", err)
	}

	log.Printf("[MAIN] Created PID file %s with PID %d", PID_FILE, currentPid)
	return nil
}

// removePidFile removes the PID file on shutdown
func removePidFile() {
	if err := os.Remove(PID_FILE); err != nil {
		log.Printf("[MAIN] Warning: failed to remove PID file: %v", err)
	} else {
		log.Printf("[MAIN] Removed PID file %s", PID_FILE)
	}
}

func initContext() *librecashContext.Context {
	log.Println("[MAIN] Initializing application context")

	log.Printf("[MAIN] Using Telegram token: %s...", config.C().Telegram_Token[:10])
	log.Printf("[MAIN] Using database connection string: %s", config.C().Db_Conn_Str)
	log.Printf("[MAIN] Using RabbitMQ URL: %s", config.C().Rabbit_Url)

	appContext := &librecashContext.Context{}

	// Initialize Telegram Bot
	log.Println("[MAIN] Connecting to Telegram Bot API...")
	bot, err := tgbotapi.NewBotAPI(config.C().Telegram_Token)
	if err != nil {
		log.Fatalf("[MAIN] Failed to connect to Telegram: %v", err)
	}
	log.Printf("[MAIN] Authorized on Telegram account: %s", bot.Self.UserName)

	// Initialize Database
	log.Println("[MAIN] Connecting to PostgreSQL database...")
	db, err := sql.Open("postgres", config.C().Db_Conn_Str)
	if err != nil {
		log.Fatalf("[MAIN] Failed to open database connection: %v", err)
	}

	// Test database connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("[MAIN] Failed to ping database: %v", err)
	}
	log.Println("[MAIN] Successfully connected to the database")

	// Set up context
	appContext.SetBot(bot)
	appContext.Repo = repository.NewRepository(db)
	appContext.Config = config.C()

	return appContext
}

// Message producer - handles incoming Telegram updates
func main1() {
	log.Println("[MAIN1] Starting message producer goroutine")

	appContext := initContext()
	appContext.RabbitPublish = rabbit.NewRabbitClient(config.C().Rabbit_Url, "messages")

	// Configure updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.Limit = 99

	log.Println("[MAIN1] Starting to receive Telegram updates...")
	updates, err := appContext.GetBot().GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("[MAIN1] Failed to get updates channel: %v", err)
	}

	log.Println("[MAIN1] Message producer ready, waiting for messages...")

	for update := range updates {
		// Handle regular messages
		if update.Message != nil {
			// Ignore messages from chats (not from users)
			if update.Message.From == nil {
				log.Println("[MAIN1] Ignoring message without From field (probably from a channel)")
				continue
			}

			userId := update.Message.Chat.ID
			startTime := time.Now()

			log.Printf("[MAIN1] Received message from user %d (@%s): %s",
				userId, update.Message.From.UserName, update.Message.Text)

			// Handle the message
			menu.HandleMessage(appContext, userId, update.Message)

			duration := time.Since(startTime)
			log.Printf("[MAIN1] Message processing completed for user %d (total duration: %v)", userId, duration)
		}

		// Handle callback queries (inline button presses)
		if update.CallbackQuery != nil {
			userId := int64(update.CallbackQuery.From.ID)
			log.Printf("[MAIN1] Received callback from user %d: %s", userId, update.CallbackQuery.Data)

			// Handle the callback
			menu.HandleCallback(appContext, userId, update.CallbackQuery)
		}
	}
}

// Message consumer - sends messages to Telegram with rate limiting
func main2() {
	log.Println("[MAIN2] Starting message consumer goroutine")

	appContext := initContext()
	appContext.RabbitConsume = rabbit.NewRabbitClient(config.C().Rabbit_Url, "messages")

	// Create and start sender
	s := sender.NewSender(appContext)
	s.Start()

	log.Println("[MAIN2] Message consumer ready")
}

var (
	isShuttingDown bool
)

func setupGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, starting graceful shutdown", sig)
		gracefulShutdown()
		os.Exit(0)
	}()
}

func gracefulShutdown() {
	log.Println("Starting graceful shutdown (max 30 seconds)")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	isShuttingDown = true

	// Wait for current operations to complete (simplified)
	log.Println("Waiting for operations to complete...")

	// In a real implementation, we would:
	// - Wait for main1 and main2 to finish current operations
	// - Close database connections gracefully
	// - Close RabbitMQ connections
	// - Flush remaining metrics

	// For now, just wait a bit to simulate graceful shutdown
	select {
	case <-time.After(2 * time.Second):
		log.Println("Operations completed")
	case <-ctx.Done():
		log.Println("Timeout reached, forcing shutdown")
	}

	log.Println("Graceful shutdown completed")
}

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Initialize configuration
	config.Init("librecash")

	// Create PID file to prevent multiple instances
	if err := createPidFile(); err != nil {
		log.Fatalf("[MAIN] %v", err)
	}

	// Ensure PID file is removed on exit
	defer removePidFile()

	// Initialize metrics system (PRD022)
	if err := metrics.Init(); err != nil {
		log.Fatalf("[MAIN] Failed to initialize metrics: %v", err)
	}

	// Setup graceful shutdown
	setupGracefulShutdown()

	log.Println("[MAIN] Starting LibreCash bot...")
	log.Println("[MAIN] Press Ctrl+C to stop")

	// Start producer and consumer in separate goroutines
	go main1()
	go main2()

	// Keep the main goroutine alive
	forever := make(chan bool)
	<-forever
}
