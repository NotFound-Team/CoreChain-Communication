package chat

import (
	"context"
	"log"

	"corechain-communication/internal/client"
	"corechain-communication/internal/db"
	"corechain-communication/internal/storage"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MemberDetail struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Role   string `json:"role"`
}

type ConversationDetail struct {
	ID                    int64             `json:"id"`
	Name                  string            `json:"name,omitempty"`
	Avatar                string            `json:"avatar,omitempty"`
	IsGroup               bool              `json:"is_group"`
	Members               []MemberDetail    `json:"members"`
	Messages              []MessageResponse `json:"messages"`
	LastMessageID         int64             `json:"last_message_id"`
	LastMessageAt         pgtype.Timestamp  `json:"last_message_at"`
	LastMessageContent    string            `json:"last_message_content"`
	LastMessageSenderID   string            `json:"last_message_sender_id"`
	LastMessageSenderName string            `json:"last_message_sender_name"`
	LastMessageType       string            `json:"last_message_type"`
	LastMessageFileName   string            `json:"last_message_file_name,omitempty"`
	CreatedAt             pgtype.Timestamp  `json:"created_at"`
	UpdatedAt             pgtype.Timestamp  `json:"updated_at"`
}

type ConversationSummary struct {
	ID                    int64            `json:"id"`
	Name                  string           `json:"name"`
	Avatar                string           `json:"avatar"`
	IsGroup               bool             `json:"is_group"`
	LastMessageID         int64            `json:"last_message_id"`
	LastMessageAt         pgtype.Timestamp `json:"last_message_at"`
	LastMessageContent    string           `json:"last_message_content"`
	LastMessageSenderID   string           `json:"last_message_sender_id"`
	LastMessageSenderName string           `json:"last_message_sender_name"`
	LastMessageType       string           `json:"last_message_type"`
	LastMessageFileName   string           `json:"last_message_file_name,omitempty"`
	LastReadMessageID     int64            `json:"last_read_message_id"`
	UnreadCount           int64            `json:"unread_count"`
}

type MessageResponse struct {
	db.Message
	FileURL string `json:"file_url"`
}

type ChatService struct {
	queries    *db.Queries
	pool       *pgxpool.Pool
	userClient *client.UserClient
}

func NewChatService(q *db.Queries, p *pgxpool.Pool, uc *client.UserClient) *ChatService {
	return &ChatService{
		queries:    q,
		pool:       p,
		userClient: uc,
	}
}

func (s *ChatService) GetOrCreatePrivateConversation(ctx context.Context, userID, partnerID string) (int64, error) {
	convID, err := s.queries.GetPrivateConversation(ctx, db.GetPrivateConversationParams{
		UserID:   userID,
		UserID_2: partnerID,
	})

	if err == nil {
		return convID, nil
	}

	// Fetch partner info to ensure they exist
	_, err = s.userClient.GetSingleUser(ctx, partnerID)
	if err != nil {
		return 0, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	newConv, err := qtx.CreateConversation(ctx, db.CreateConversationParams{
		Name:    pgtype.Text{Valid: false},
		Avatar:  pgtype.Text{Valid: false},
		IsGroup: pgtype.Bool{Bool: false, Valid: true},
	})
	if err != nil {
		return 0, err
	}

	participants := []string{userID, partnerID}
	for _, pid := range participants {
		err = qtx.AddParticipant(ctx, db.AddParticipantParams{
			ConversationID: newConv.ID,
			UserID:         pid,
			Role:           pgtype.Text{String: "member", Valid: true},
		})
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	// TODO: Sync Redis cache here db.CacheParticipants

	return newConv.ID, nil
}

func (s *ChatService) ListConversations(ctx context.Context, userID string, limit, offset int32) ([]ConversationSummary, error) {
	rows, err := s.queries.ListConversationsByUser(ctx, db.ListConversationsByUserParams{
		SenderID: userID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	// 1. Collect all participant IDs to batch fetch
	idSet := make(map[string]bool)
	for _, r := range rows {
		// If private, we need the partner's info
		if !r.IsGroup.Bool {
			for _, pid := range r.ParticipantIds {
				if pid != userID {
					idSet[pid] = true
				}
			}
		}
	}

	userIDs := make([]string, 0, len(idSet))
	for id := range idSet {
		userIDs = append(userIDs, id)
	}

	// 2. Enrich
	// We also need the sender of the last message in each conversation
	lastMsgSenderIDs := make([]string, 0)
	for _, r := range rows {
		if r.LastMessageSenderID.String != "" {
			lastMsgSenderIDs = append(lastMsgSenderIDs, r.LastMessageSenderID.String)
		}
	}
	userIDs = append(userIDs, lastMsgSenderIDs...)

	userMap, err := s.userClient.EnrichUsers(ctx, userIDs)
	if err != nil {
		// log error but continue with empty map
	}
	if userMap == nil {
		userMap = make(map[string]client.UserInfo)
	}

	// 3. Build result
	var result []ConversationSummary
	for _, r := range rows {
		name := r.Name.String
		avatar := r.Avatar.String

		if !r.IsGroup.Bool {
			// Find partner
			var partnerID string
			for _, pid := range r.ParticipantIds {
				if pid != userID {
					partnerID = pid
					break
				}
			}
			if u, ok := userMap[partnerID]; ok {
				name = u.Name
				avatar = u.Avatar
			}
		}

		lastMessageSenderName := ""
		if u, ok := userMap[r.LastMessageSenderID.String]; ok {
			lastMessageSenderName = u.Name
		}

		result = append(result, ConversationSummary{
			ID:                    r.ID,
			Name:                  name,
			Avatar:                avatar,
			IsGroup:               r.IsGroup.Bool,
			LastMessageID:         r.LastMessageID.Int64,
			LastMessageAt:         r.LastMessageAt,
			LastMessageContent:    r.LastMessageContent.String,
			LastMessageSenderID:   r.LastMessageSenderID.String,
			LastMessageSenderName: lastMessageSenderName,
			LastReadMessageID:     r.LastReadMessageID.Int64,
			LastMessageType:       r.LastMessageType.String,
			LastMessageFileName:   r.LastMessageFileName.String,
			UnreadCount:           r.UnreadCount,
		})
	}

	return result, nil
}

func (s *ChatService) GetMessages(ctx context.Context, convID int64, limit int32, beforeID int64) ([]MessageResponse, error) {
	dbMessages, err := s.queries.GetMessagesByConversation(ctx, db.GetMessagesByConversationParams{
		ConversationID: convID,
		BeforeID:       beforeID,
		LimitCount:     limit,
	})
	if err != nil {
		return nil, err
	}

	finalMessages := make([]MessageResponse, len(dbMessages))
	for i, m := range dbMessages {
		res := MessageResponse{
			Message: m,
		}

		if m.Type.String == "file" && m.FilePath.String != "" {
			signedURL, err := storage.GetPresignedURL(m.FilePath.String)
			if err != nil {
				log.Printf("Error signing URL for historical message %d: %v", m.ID, err)
			} else {
				res.FileURL = signedURL
			}
		}
		finalMessages[i] = res
	}

	return finalMessages, nil
}

func (s *ChatService) GetConversation(ctx context.Context, conversationID int64) (*ConversationDetail, error) {
	conv, err := s.queries.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	participants, err := s.queries.ListParticipantsByConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, len(participants))
	for i, p := range participants {
		userIDs[i] = p.UserID
	}

	userMap, err := s.userClient.EnrichUsers(ctx, userIDs)
	if err != nil {
		log.Printf("Warning: failed to enrich some users: %v", err)
	}
	if userMap == nil {
		userMap = make(map[string]client.UserInfo)
	}

	members := make([]MemberDetail, len(participants))
	for i, p := range participants {
		uInfo := userMap[p.UserID]
		members[i] = MemberDetail{
			UserID: p.UserID,
			Name:   uInfo.Name,
			Avatar: uInfo.Avatar,
			Role:   p.Role.String,
		}
	}

	name := conv.Name.String
	avatar := conv.Avatar.String

	if !conv.IsGroup.Bool {
		var currentUserID string
		if v := ctx.Value("user_id"); v != nil {
			currentUserID, _ = v.(string)
		}

		if currentUserID != "" {
			for _, p := range participants {
				if p.UserID != currentUserID {
					if u, ok := userMap[p.UserID]; ok {
						name = u.Name
						avatar = u.Avatar
					}
					break
				}
			}
		}
	}

	dbMessages, err := s.queries.GetMessagesByConversation(ctx, db.GetMessagesByConversationParams{
		ConversationID: conversationID,
		BeforeID:       0,
		LimitCount:     20,
	})
	if err != nil {
		return nil, err
	}

	finalMessages := make([]MessageResponse, len(dbMessages))

	for i, msg := range dbMessages {
		res := MessageResponse{
			Message: msg,
		}
		if msg.Type.String == "file" && msg.FilePath.String != "" {
			signedURL, err := storage.GetPresignedURL(msg.FilePath.String)
			if err != nil {
				log.Printf("Error signing URL for message %d: %v", msg.ID, err)
			} else {
				res.FileURL = signedURL
			}
		}

		finalMessages[i] = res
	}

	lastMessageSenderName := ""
	if u, ok := userMap[conv.LastMessageSenderID.String]; ok {
		lastMessageSenderName = u.Name
	}

	return &ConversationDetail{
		ID:                    conv.ID,
		Name:                  name,
		Avatar:                avatar,
		IsGroup:               conv.IsGroup.Bool,
		Members:               members,
		Messages:              finalMessages,
		LastMessageID:         conv.LastMessageID.Int64,
		LastMessageAt:         conv.LastMessageAt,
		LastMessageContent:    conv.LastMessageContent.String,
		LastMessageSenderID:   conv.LastMessageSenderID.String,
		LastMessageSenderName: lastMessageSenderName,
		LastMessageType:       conv.LastMessageType.String,
		LastMessageFileName:   conv.LastMessageFileName.String,
		CreatedAt:             conv.CreatedAt,
		UpdatedAt:             conv.UpdatedAt,
	}, nil
}
