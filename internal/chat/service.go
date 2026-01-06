package chat

import (
	"context"

	"corechain-communication/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewChatService(q *db.Queries, p *pgxpool.Pool) *ChatService {
	return &ChatService{
		queries: q,
		pool:    p,
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

func (s *ChatService) ListConversations(ctx context.Context, userID string, limit, offset int32) ([]db.ListConversationsByUserRow, error) {
	return s.queries.ListConversationsByUser(ctx, db.ListConversationsByUserParams{
		SenderID: userID,
		Limit:    limit,
		Offset:   offset,
	})
}

func (s *ChatService) GetMessages(ctx context.Context, convID int64, limit int32, beforeID int64) ([]db.Message, error) {
	return s.queries.GetMessagesByConversation(ctx, db.GetMessagesByConversationParams{
		ConversationID: convID,
		Limit:          limit,
	})
}
