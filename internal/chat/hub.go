package chat

import (
	"context"
	"corechain-communication/internal/broker"
	"corechain-communication/internal/config"
	"corechain-communication/internal/db"
	"encoding/json"
	"log"
	"strconv"
	"sync"
)

type Message struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	Content        string `json:"content"`
}

type Hub struct {
	clients map[string]*Client

	register chan *Client

	unregister chan *Client

	broadcast chan []byte

	mu sync.RWMutex

	q *db.Queries
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
		return
	}

	memberIDs, err := db.GetCachedParticipants(ctx, msg.ConversationID)

	if err != nil || len(memberIDs) == 0 {
		convIDInt, _ := strconv.ParseInt(msg.ConversationID, 10, 64)

		rows, err := h.q.ListParticipantsByConversation(ctx, convIDInt)
		if err == nil {
			for _, r := range rows {
				memberIDs = append(memberIDs, r.UserID)
			}
			db.CacheParticipants(ctx, msg.ConversationID, memberIDs)
		}
	}

	for _, memberID := range memberIDs {
		if memberID == msg.SenderID {
			continue
		}

		h.mu.RLock()
		client, online := h.clients[memberID]
		h.mu.RUnlock()

		if online {
			select {
			case client.Send <- rawData:
			default:
				h.unregister <- client
				h.sendToPushTopic(ctx, memberID, msg)
			}
		} else {
			h.sendToPushTopic(ctx, memberID, msg)
		}
		_ = broker.Get().PushEvent(ctx, config.Get().KafkaTopicPersistence, msg.ConversationID, msg)
	}
}

func (h *Hub) sendToPushTopic(ctx context.Context, userID string, msg Message) {
	pushPayload := map[string]interface{}{
		"receiver_id": userID,
		"content":     msg.Content,
		"type":        msg.Type,
	}
	_ = broker.Get().PushEvent(ctx, config.Get().KafkaTopicNotification, userID, pushPayload)
}
