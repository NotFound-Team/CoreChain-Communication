package meeting

import (
	"context"
	"corechain-communication/internal/config"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

type LiveKitService struct {
	apiKey    string
	apiSecret string
	serverURL string
	client    *lksdk.RoomServiceClient
}

func NewLiveKitService() *LiveKitService {
	cfg := config.Get()

	roomClient := lksdk.NewRoomServiceClient(cfg.LiveKitURL, cfg.LiveKitAPIKey, cfg.LiveKitAPISecret)

	return &LiveKitService{
		apiKey:    cfg.LiveKitAPIKey,
		apiSecret: cfg.LiveKitAPISecret,
		serverURL: cfg.LiveKitURL,
		client:    roomClient,
	}
}

func (s *LiveKitService) GenerateToken(roomName, identity, participantName, meetingTitle string) (string, error) {
	at := auth.NewAccessToken(s.apiKey, s.apiSecret)

	canPublish := true
	canSubscribe := true
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         roomName,
		CanPublish:   &canPublish,
		CanSubscribe: &canSubscribe,
	}

	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(participantName).
		SetMetadata(meetingTitle).
		SetValidFor(time.Hour * 2)

	return at.ToJWT()
}

func (s *LiveKitService) DeleteRoom(ctx context.Context, roomName string) error {
	_, err := s.client.DeleteRoom(ctx, &livekit.DeleteRoomRequest{
		Room: roomName,
	})
	return err
}
