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
	"corechain-communication/internal/meeting"
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
	db.RunMigration(cfg.MigrationURL, cfg.DatabaseURL)
	db.InitRedis()
	broker.InitKafka()
	defer broker.Get().Close()

	queries := db.New(pool)
	hub := chat.NewHub(queries)
	go hub.Run()
	go worker.StartDBWorker(cfg, queries)

	userClient := client.NewUserClient(cfg.UserServiceURL)
	chatService := chat.NewChatService(queries, pool, userClient)

	lkService := meeting.NewLiveKitService()
	meetingService := meeting.NewMeetingService(pool, queries, lkService)

	meetingHandler := meeting.NewMeetingHandler(meetingService)
	chatHandler := chat.NewHandler(hub, chatService)

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", chatHandler.ServeWS)

	mux.HandleFunc("/upload", middleware.WithAuth(storage.UploadHandler))

	mux.HandleFunc("/conversations/private", middleware.WithAuth(chatHandler.HandleGetOrCreatePrivateConv))
	mux.HandleFunc("/conversations/detail", middleware.WithAuth(chatHandler.HandleGetConversation))
	mux.HandleFunc("/conversations/unread-count", middleware.WithAuth(chatHandler.HandleGetUnreadCount))
	mux.HandleFunc("/conversations", middleware.WithAuth(chatHandler.HandleListConversations))

	mux.HandleFunc("/messages", middleware.WithAuth(chatHandler.HandleGetMessages))

	mux.HandleFunc("/meetings/my", middleware.WithAuth(meetingHandler.ListMyMeetings))
	mux.HandleFunc("/meetings/join", middleware.WithAuth(meetingHandler.JoinMeeting))
	mux.HandleFunc("/meetings/end", middleware.WithAuth(meetingHandler.EndMeeting))
	mux.HandleFunc("/meetings", middleware.WithAuth(meetingHandler.CreateMeeting))

	handlerWithCORS := middleware.EnableCORS(mux)

	log.Println("Server started on port", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, handlerWithCORS))
}
