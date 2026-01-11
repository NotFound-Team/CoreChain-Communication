package main

import (
	"context"
	"log"
	"net/http"

	"corechain-communication/internal/broker"
	"corechain-communication/internal/chat"
	"corechain-communication/internal/client"
	"corechain-communication/internal/config"
	"corechain-communication/internal/db"
	"corechain-communication/internal/middleware"
	"corechain-communication/internal/storage"
	"corechain-communication/internal/worker"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	storage.InitMinio()

	queries := db.New(pool)
	userClient := client.NewUserClient(cfg.UserServiceURL)
	chatService := chat.NewChatService(queries, pool, userClient)
	hub := chat.NewHub(queries)

	db.RunMigration(cfg.MigrationURL, cfg.DatabaseURL)
	db.InitRedis()
	broker.InitKafka()
	defer broker.Get().Close()

	go hub.Run()
	go worker.StartDBWorker(cfg, queries)

	chatHandler := chat.NewHandler(hub, chatService)
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", chatHandler.ServeWS)
	mux.HandleFunc("/upload", middleware.WithAuth(storage.UploadHandler))
	mux.HandleFunc("/conversations/private", middleware.WithAuth(chatHandler.HandleGetOrCreatePrivateConv))
	mux.HandleFunc("/conversations/detail", middleware.WithAuth(chatHandler.HandleGetConversation))
	mux.HandleFunc("/conversations", middleware.WithAuth(chatHandler.HandleListConversations))
	mux.HandleFunc("/messages", middleware.WithAuth(chatHandler.HandleGetMessages))

	handlerWithCORS := middleware.EnableCORS(mux)

	log.Println("Server started on port", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, handlerWithCORS))
}
