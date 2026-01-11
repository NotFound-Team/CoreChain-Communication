package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type UserInfo struct {
	ID     string `json:"_id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type UserResponse struct {
	StatusCode int      `json:"statusCode"`
	Message    string   `json:"message"`
	Data       UserInfo `json:"data"`
}

type UserClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewUserClient(baseURL string) *UserClient {
	return &UserClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *UserClient) GetUsersBatch(ctx context.Context, ids []string) (map[string]UserInfo, error) {
	if len(ids) == 0 {
		return make(map[string]UserInfo), nil
	}

	payload := map[string][]string{"ids": ids}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/users/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service error: %d", resp.StatusCode)
	}

	var result []UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	userMap := make(map[string]UserInfo)
	for _, u := range result {
		userMap[u.ID] = u
	}

	return userMap, nil
}

func (c *UserClient) EnrichUsers(ctx context.Context, ids []string) (map[string]UserInfo, error) {
	/*
	   return c.GetUsersBatch(ctx, ids)
	*/

	userMap := make(map[string]UserInfo)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, id := range ids {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()

			info, err := c.GetSingleUser(ctx, userID)
			if err == nil {
				mu.Lock()
				userMap[userID] = info
				mu.Unlock()
			}
		}(id)
	}

	wg.Wait()
	return userMap, nil
}

func (c *UserClient) GetSingleUser(ctx context.Context, id string) (UserInfo, error) {
	url := fmt.Sprintf("%s/users/public/%s", c.BaseURL, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return UserInfo{}, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return UserInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UserInfo{}, fmt.Errorf("user service error: %d", resp.StatusCode)
	}

	var wrapper UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return UserInfo{}, fmt.Errorf("decode error: %w", err)
	}
	log.Println("user info:", wrapper.Data)
	return wrapper.Data, nil
}
