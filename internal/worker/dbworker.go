package worker

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

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
		MinBytes: 10e3,
		MaxBytes: 10e6,
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

		convID, _ := strconv.ParseInt(msg.ConversationID, 10, 64)

		params := db.CreateMessageParams{
			ConversationID: convID,
			SenderID:       msg.SenderID,
			Content:        pgtype.Text{String: msg.Content, Valid: true},
			Type:           pgtype.Text{String: msg.Type, Valid: true},
			ReplyToID:      pgtype.Int8{Valid: false},
		}

		_, err = q.CreateMessage(context.Background(), params)
		if err != nil {
			log.Printf("DB Save Error: %v", err)
			continue
		}

		log.Printf("Persisted [%s] from %s to Conv %d", msg.Type, msg.SenderID, convID)
	}
}
