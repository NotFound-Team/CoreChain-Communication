package db

import (
	"context"
	"corechain-communication/internal/config"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	redisOnce   sync.Once
)

const (
	UserStatusTTL  = 0              // Online status never expired
	UserProfileTTL = 24 * time.Hour // Cache user info in one day
)

func InitRedis() {
	redisOnce.Do(func() {
		cfg := config.Get()
		redisClient = redis.NewClient(&redis.Options{
			Addr: cfg.RedisAddr,
		})
	})
}

func GetRedis() *redis.Client {
	return redisClient
}

func SetUserOnline(ctx context.Context, userID string) error {
	return redisClient.Set(ctx, "online:"+userID, "true", UserStatusTTL).Err()
}

// SetUserOffline when user disconnected
func SetUserOffline(ctx context.Context, userID string) error {
	return redisClient.Del(ctx, "online:"+userID).Err()
}

// IsUserOnline check is user online
func IsUserOnline(ctx context.Context, userID string) bool {
	val, err := redisClient.Get(ctx, "online:"+userID).Result()
	return err == nil && val == "true"
}

func CacheUserProfile(ctx context.Context, userID string, profileData []byte) error {
	return redisClient.Set(ctx, "profile:"+userID, profileData, UserProfileTTL).Err()
}

func GetCachedUserProfile(ctx context.Context, userID string) ([]byte, error) {
	return redisClient.Get(ctx, "profile:"+userID).Bytes()
}

// CacheParticipants save UserID into Redis
func CacheParticipants(ctx context.Context, convID string, userIDs []string) error {
	key := "conv_members:" + convID
	redisClient.Del(ctx, key)
	return redisClient.SAdd(ctx, key, userIDs).Err()
}

// GetCachedParticipants
func GetCachedParticipants(ctx context.Context, convID string) ([]string, error) {
	key := "conv_members:" + convID
	return redisClient.SMembers(ctx, key).Result()
}
