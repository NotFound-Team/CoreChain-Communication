package meeting

import (
	"math/rand"
	"time"
)

func GenerateMeetingKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[seed.Intn(len(charset))]
	}
	return string(b[0:3]) + "-" + string(b[3:6])
}
