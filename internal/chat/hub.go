package chat

import (
	"context"
	"corechain-communication/internal/broker"
	"corechain-communication/internal/config"
	"corechain-communication/internal/db"
	"corechain-communication/internal/storage"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"
)

type Message struct {
	ID             int64  `json:"id,omitempty"`
	ClientMsgID    string `json:"client_msg_id,omitempty"`
	Type           string `json:"type"`
	ConversationID int64  `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	SenderName     string `json:"sender_name,omitempty"`
	Content        string `json:"content"`

	FileName string `json:"file_name,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	FilePath string `json:"file_path,omitempty"`
	FileURL  string `json:"file_url,omitempty"`
	FileType string `json:"file_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	LastReadMessageID int64 `json:"last_read_message_id,omitempty"`
}

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
	q          *db.Queries
}

func NewHub(q *db.Queries) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		q:          q,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("User %s connected", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				close(client.Send)
				log.Printf("User %s disconnected", client.UserID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.handleMessageDelivery(message)
		}
	}
}

func (h *Hub) handleMessageDelivery(rawData []byte) {
	ctx := context.Background()
	var msg Message
	if err := json.Unmarshal(rawData, &msg); err != nil {
		log.Println("failed to unmarshal message: ", err)
		return
	}

	kafkaKey := strconv.FormatInt(msg.ConversationID, 10)
	err := broker.Get().PushEvent(ctx, config.Get().KafkaTopicPersistence, kafkaKey, msg)
	if err != nil {
		log.Printf("Failed to push event persistence for Conv %d: %v", msg.ConversationID, err)
	} else {
		log.Printf("Pushed message to Kafka persistence: %s", msg.Content)
	}

	if msg.Type == "mark_as_read" {
		log.Printf("Received mark_as_read for Conv %d from %s", msg.ConversationID, msg.SenderID)
	} else if msg.Type == "file" && msg.FilePath != "" {
		signedURL, err := storage.GetPresignedURL(msg.FilePath)
		if err == nil {
			msg.FileURL = signedURL
			newRawData, err := json.Marshal(msg)
			if err == nil {
				rawData = newRawData
			}
		} else {
			log.Printf("Error signing URL in Hub for file %s: %v", msg.FilePath, err)
		}
	}

	convIDStr := strconv.FormatInt(msg.ConversationID, 10)
	memberIDs, err := db.GetCachedParticipants(ctx, convIDStr)

	if err != nil || len(memberIDs) == 0 {
		log.Println("Cache miss for participants, fetching from DB...")
		rows, err := h.q.ListParticipantsByConversation(ctx, msg.ConversationID)
		if err == nil {
			for _, r := range rows {
				memberIDs = append(memberIDs, r.UserID)
			}
			db.CacheParticipants(ctx, convIDStr, memberIDs)
		}
	}

	for _, memberID := range memberIDs {
		h.mu.RLock()
		client, online := h.clients[memberID]
		h.mu.RUnlock()

		if online {
			select {
			case client.Send <- rawData:
				log.Printf("Delivered message to %s (Online)", memberID)
			default:
				h.unregister <- client
				if memberID != msg.SenderID {
					h.sendToPushTopic(ctx, memberID, msg)
				}
			}
		} else {
			if memberID != msg.SenderID {
				h.sendToPushTopic(ctx, memberID, msg)
			}
		}
	}
}

func (h *Hub) sendToPushTopic(ctx context.Context, userID string, msg Message) {
	pushPayload := map[string]interface{}{
		"receiver_id": userID,
		"content":     msg.Content,
		"type":        msg.Type,
		"sender_id":   msg.SenderID,
		"sender_name": msg.SenderName,
	}
	_ = broker.Get().PushEvent(ctx, config.Get().KafkaTopicNotification, userID, pushPayload)
}
