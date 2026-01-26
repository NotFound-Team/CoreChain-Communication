package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"corechain-communication/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

type Handler struct {
	hub     *Hub
	service *ChatService
}

func NewHandler(h *Hub, s *ChatService) *Handler {
	return &Handler{
		hub:     h,
		service: s,
	}
}

// =======================
// 1. WebSocket Handler
// =======================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "Unauthorized: Token required", http.StatusUnauthorized)
		return
	}

	// Validate JWT
	claims, err := validateToken(tokenString)
	if err != nil {
		log.Printf("WS Auth Error: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := fmt.Sprintf("%v", claims["_id"])

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{
		UserID: userID,
		Hub:    h.hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	h.hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

// =======================
// 2. REST API Handlers
// =======================

// POST /conversations/private
func (h *Handler) HandleGetOrCreatePrivateConv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value("user_id").(string)

	var req struct {
		PartnerID string `json:"partner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	convID, err := h.service.GetOrCreatePrivateConversation(r.Context(), userID, req.PartnerID)
	if err != nil {
		log.Printf("Error creating conv: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	convDetail, err := h.service.GetConversation(r.Context(), convID)
	if err != nil {
		log.Printf("Error getting conv: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	log.Println("conv detail: ", convDetail)
	jsonResponse(w, convDetail)
}

// GET /conversations
func (h *Handler) HandleListConversations(w http.ResponseWriter, r *http.Request) {
	log.Println("handle list conversation")
	userID := r.Context().Value("user_id").(string)

	// Parse query params (limit, offset)
	limit := parseQueryInt(r, "limit", 20)
	offset := parseQueryInt(r, "offset", 0)

	convs, err := h.service.ListConversations(r.Context(), userID, int32(limit), int32(offset))
	log.Println("conversations: ", convs)
	if err != nil {
		log.Printf("error: %v\n", err)
		http.Error(w, "Failed to fetch conversations", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, convs)
}

// GET /conversations/messages?conversation_id=123
func (h *Handler) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	convIDStr := r.URL.Query().Get("conversation_id")
	convID, _ := strconv.ParseInt(convIDStr, 10, 64)

	beforeIDStr := r.URL.Query().Get("before_id")
	beforeID, _ := strconv.ParseInt(beforeIDStr, 10, 64)

	limit := parseQueryInt(r, "limit", 20)

	msgs, err := h.service.GetMessages(r.Context(), convID, int32(limit), beforeID)
	if err != nil {
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, msgs)
}

// GET /conversations/detail?id=123
func (h *Handler) HandleGetConversation(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	convID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	convDetail, err := h.service.GetConversation(r.Context(), convID)
	if err != nil {
		log.Printf("Error fetching conversation detail: %v", err)
		http.Error(w, "Failed to fetch conversation", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, convDetail)
}

// =======================
// Helpers
// =======================

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": data,
	})
}

func parseQueryInt(r *http.Request, key string, defaultVal int) int {
	valStr := r.URL.Query().Get(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func validateToken(tokenString string) (jwt.MapClaims, error) {
	jwtSecret := config.Get().JwtSecret
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected method: %v", t.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, errors.New("invalid claims")
}
