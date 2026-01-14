package meeting

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"corechain-communication/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MeetingService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	lk      *LiveKitService
}

func NewMeetingService(pool *pgxpool.Pool, queries *db.Queries, lk *LiveKitService) *MeetingService {
	return &MeetingService{
		pool:    pool,
		queries: queries,
		lk:      lk,
	}
}

func (s *MeetingService) CreateMeeting(ctx context.Context, title, desc, hostID string, invitedIDs []string, startTime time.Time) (*db.Meeting, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %v", err)
	}

	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	var meeting db.Meeting

	success := false
	for i := 0; i < 5; i++ {
		mKey := GenerateMeetingKey()
		roomName := uuid.New().String()

		meeting, err = qtx.CreateMeeting(ctx, db.CreateMeetingParams{
			Title:       title,
			Description: pgtype.Text{String: desc, Valid: desc != ""},
			HostID:      hostID,
			RoomName:    roomName,
			MeetingKey:  mKey,
			StartTime:   pgtype.Timestamptz{Time: startTime, Valid: true},
		})

		if err == nil {
			success = true
			break
		}
	}

	if !success {
		return nil, fmt.Errorf("failed to create meeting after retries: %v", err)
	}

	for _, uID := range invitedIDs {
		err := qtx.AddMeetingInvite(ctx, db.AddMeetingInviteParams{
			MeetingID: meeting.ID,
			UserID:    uID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add invite for user %s: %v", uID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("could not commit transaction: %v", err)
	}

	return &meeting, nil
}

func (s *MeetingService) JoinMeeting(ctx context.Context, userID, userName, roomName, meetingKey string) (string, string, error) {
	var meeting db.Meeting
	var err error

	if meetingKey != "" {
		meeting, err = s.queries.GetActiveMeetingByKey(ctx, meetingKey)
		if err != nil {
			return "", "", errors.New("invalid meeting key or meeting was ended")
		}
	} else if roomName != "" {
		meeting, err = s.queries.GetMeetingByRoomName(ctx, roomName)
		if err != nil {
			return "", "", errors.New("meeting not found")
		}
		hasPerm, err := s.queries.CheckJoinPermission(ctx, db.CheckJoinPermissionParams{
			RoomName: meeting.RoomName,
			UserID:   userID,
		})
		if err != nil || !hasPerm {
			return "", "", errors.New("you do not have permission to join this room")
		}
	} else {
		return "", "", errors.New("room_name or meeting_key is required")
	}

	isHost := (meeting.HostID == userID)
	now := time.Now()

	if !isHost {
		if now.Before(meeting.StartTime.Time) {
			if !meeting.IsActive.Bool {
				return "", "", fmt.Errorf("meeting is not active yet, please join at %s",
					meeting.StartTime.Time.Format("HH:mm dd/MM/yyyy"))
			}
		}
	}

	if isHost && !meeting.IsActive.Bool {
		err := s.queries.UpdateMeetingStatus(ctx, db.UpdateMeetingStatusParams{
			ID:       meeting.ID,
			IsActive: pgtype.Bool{Bool: true, Valid: true},
		})
		if err != nil {
			log.Printf("Warning: could not activate meeting: %v\n", err)
		}
	}

	token, err := s.lk.GenerateToken(meeting.RoomName, userID, userName, meeting.Title)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate LiveKit token: %v", err)
	}

	return token, meeting.RoomName, nil
}

func (s *MeetingService) EndMeeting(ctx context.Context, userID, roomName string) error {
	meeting, err := s.queries.EndMeeting(ctx, db.EndMeetingParams{
		RoomName: roomName,
		HostID:   userID,
	})
	if err != nil {
		return errors.New("you do not have permission to end this meeting or meeting was already ended")
	}

	err = s.lk.DeleteRoom(ctx, meeting.RoomName)

	if err != nil {
		fmt.Printf("LiveKit DeleteRoom Warning: %v\n", err)
	}

	return nil
}

func (s *MeetingService) ListMyMeetings(ctx context.Context, userID string) ([]db.Meeting, error) {
	meetings, err := s.queries.ListMyMeetings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list your meetings: %v", err)
	}
	return meetings, nil
}
