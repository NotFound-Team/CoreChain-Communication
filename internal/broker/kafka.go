package broker

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"corechain-communication/internal/config"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

var (
	instance *KafkaProducer
	once     sync.Once
)

func InitKafka() {
	once.Do(func() {
		cfg := config.Get()
		w := &kafka.Writer{
			Addr:         kafka.TCP(cfg.KafkaBroker),
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        true,
			WriteTimeout: 10 * time.Second,
		}
		instance = &KafkaProducer{writer: w}
		log.Println("Kafka Producer initialized successfully")
	})
}

func Get() *KafkaProducer {
	if instance == nil {
		panic("Kafka Producer is not initialized. Call InitKafka first.")
	}
	return instance
}

func (p *KafkaProducer) PushEvent(ctx context.Context, topic, key string, payload any) error {
	value, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	})

	if err != nil {
		log.Printf("Kafka Write Error: %v", err)
		return err
	}
	return nil
}

func (p *KafkaProducer) Close() {
	if p.writer != nil {
		p.writer.Close()
	}
}
