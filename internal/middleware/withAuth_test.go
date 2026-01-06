package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"corechain-communication/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func TestWithAuth(t *testing.T) {
	// Setup config
	os.Setenv("JWT_SECRET_KEY", "testsupersecret")
	// Trigger config load. Since it uses "once", we hope it hasn't been called with different settings in this test execution context properly.
	// NOTE: If LoadConfig was already called, this env might not be picked up if it already loaded.
	// But in a standalone `go test`, it should be fine.
	_, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	secret := []byte("testsupersecret")

	tests := []struct {
		name           string
		token          string
		headerValue    string
		expectedStatus int
		expectUserID   string
	}{
		{
			name:           "No Header",
			headerValue:    "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Format",
			headerValue:    "Basic foobar",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Token",
			headerValue:    "Bearer invalidtoken",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Valid Token",
			token: func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id": "12345",
					"exp":     time.Now().Add(time.Hour).Unix(),
				})
				s, _ := token.SignedString(secret)
				return s
			}(),
			headerValue: "Bearer " + func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id": "12345",
					"exp":     time.Now().Add(time.Hour).Unix(),
				})
				s, _ := token.SignedString(secret)
				return s
			}(),
			expectedStatus: http.StatusOK,
			expectUserID:   "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.headerValue != "" {
				req.Header.Set("Authorization", tt.headerValue)
			}

			rr := httptest.NewRecorder()

			handler := WithAuth(func(w http.ResponseWriter, r *http.Request) {
				userID, ok := r.Context().Value("user_id").(string)
				if !ok {
					t.Error("user_id not found in context")
				}
				if userID != tt.expectUserID {
					t.Errorf("expected user_id %v, got %v", tt.expectUserID, userID)
				}
				w.WriteHeader(http.StatusOK)
			})

			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}
		})
	}
}
