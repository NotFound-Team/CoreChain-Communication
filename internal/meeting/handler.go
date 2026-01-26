package meeting

import (
	"corechain-communication/internal/db"
	"net/http"
	"time"

	"encoding/json"
)

type MeetingHandler struct {
	service *MeetingService
}

func NewMeetingHandler(service *MeetingService) *MeetingHandler {
	return &MeetingHandler{
		service: service,
	}
}

func (h *MeetingHandler) CreateMeeting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID, _ := r.Context().Value("user_id").(string)
	role, _ := r.Context().Value("user_role").(string)

	if role != "MANAGER" && role != "ADMIN" {
		h.renderJSON(w, http.StatusForbidden, map[string]string{"error": "Manager role required"})
		return
	}

	var req struct {
		Title          string     `json:"title" binding:"required"`
		Description    string     `json:"description"`
		InvitedUserIDs []string   `json:"invited_user_ids"`
		StartTime      *time.Time `json:"start_time"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.renderJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid body"})
		return
	}

	scheduledTime := time.Now().UTC()
	if req.StartTime != nil {
		scheduledTime = *req.StartTime
	}

	meeting, err := h.service.CreateMeeting(r.Context(), req.Title, req.Description, userID, req.InvitedUserIDs, scheduledTime)
	if err != nil {
		h.renderJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	h.renderJSON(w, http.StatusCreated, meeting)
}

func (h *MeetingHandler) JoinMeeting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RoomName   string `json:"room_name"`
		MeetingKey string `json:"meeting_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.renderJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid body"})
		return
	}

	userID := r.Context().Value("user_id").(string)
	userName := r.Context().Value("user_name").(string)

	token, room, err := h.service.JoinMeeting(r.Context(), userID, userName, req.RoomName, req.MeetingKey)
	if err != nil {
		h.renderJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	h.renderJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"room_name":  room,
		"server_url": h.service.lk.serverURL,
	})
}

func (h *MeetingHandler) EndMeeting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RoomName string `json:"room_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.renderJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid body"})
		return
	}

	if req.RoomName == "" {
		h.renderJSON(w, http.StatusBadRequest, map[string]string{"error": "Room name is required"})
		return
	}

	userID := r.Context().Value("user_id").(string)

	err := h.service.EndMeeting(r.Context(), userID, req.RoomName)
	if err != nil {
		h.renderJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	h.renderJSON(w, http.StatusOK, map[string]string{"message": "Meeting ended successfully"})
}

func (h *MeetingHandler) ListMyMeetings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value("user_id").(string)

	meetings, err := h.service.ListMyMeetings(r.Context(), userID)
	if err != nil {
		h.renderJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if meetings == nil {
		meetings = []db.Meeting{}
	}

	h.renderJSON(w, http.StatusOK, meetings)
}

func (h *MeetingHandler) renderJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]interface{}{
		"data": data,
	}

	json.NewEncoder(w).Encode(response)
}
