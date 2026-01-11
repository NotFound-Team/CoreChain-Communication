package worker

import (
	"context"
	"encoding/json"
	"log"

	"corechain-communication/internal/chat"
	"corechain-communication/internal/config"
	"corechain-communication/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/segmentio/kafka-go"
)

func StartDBWorker(cfg *config.Config, q *db.Queries) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{cfg.KafkaBroker},
		Topic:    cfg.KafkaTopicPersistence,
		GroupID:  cfg.KafkaDBWorkerConsumerGroupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer reader.Close()

	log.Println("DB Worker is watching Kafka topic:", cfg.KafkaTopicPersistence)

	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Kafka Reader Error: %v", err)
			continue
		}

		var msg chat.Message
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Printf("Failed to decode message: %v", err)
			continue
		}

		params := db.CreateMessageParams{
			ConversationID: msg.ConversationID,
			SenderID:       msg.SenderID,
			Content:        pgtype.Text{String: msg.Content, Valid: msg.Content != ""},
			Type:           pgtype.Text{String: msg.Type, Valid: true},

			FileName: pgtype.Text{String: msg.FileName, Valid: msg.FileName != ""},
			FilePath: pgtype.Text{String: msg.FilePath, Valid: msg.FilePath != ""},
			FileType: pgtype.Text{String: msg.FileType, Valid: msg.FileType != ""},
			FileSize: pgtype.Int8{Int64: msg.FileSize, Valid: msg.FileSize > 0},

			ReplyToID: pgtype.Int8{Valid: false},
		}

		insertedMsg, err := q.CreateMessage(context.Background(), params)
		if err != nil {
			log.Printf("DB Save Error (Conv %d, Sender %s): %v", msg.ConversationID, msg.SenderID, err)
			continue
		}

		log.Printf("Successfully Persisted: ID=%d | Type=%s | From=%s | Conv=%d",
			insertedMsg.ID, msg.Type, msg.SenderID, msg.ConversationID)
	}
}
