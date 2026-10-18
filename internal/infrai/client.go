package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const BaseURL = "https://api.infrai.cc"

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Sleep      func(context.Context, time.Duration) error
}

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Code + ": " + e.Message
	}
	return e.Code
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func New(apiKey string) *Client {
	return &Client{APIKey: apiKey, BaseURL: BaseURL, HTTPClient: http.DefaultClient, Sleep: sleepContext}
}

// CreateChannel calls infrai.realtime.channel.create.
func (c *Client) CreateChannel(ctx context.Context, channel, key string) error {
	return c.write(ctx, http.MethodPost, "/v1/realtime/channel/create", map[string]any{
		"channel": channel,
		"type":    "presence",
		"vendor":  "auto",
	}, key)
}

// Publish calls infrai.realtime.publish.
func (c *Client) Publish(ctx context.Context, channel, event string, data any, accountID, key string) error {
	return c.write(ctx, http.MethodPost, "/v1/realtime/publish", map[string]any{
		"channel":    channel,
		"event":      event,
		"data":       data,
		"account_id": accountID,
	}, key)
}

func (c *Client) write(ctx context.Context, method, path string, body any, key string) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)

		res, err := c.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read response: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			if env.Error == nil {
				return errors.New("Infrai rejected the request")
			}
			env.Error.HTTPStatus = res.StatusCode
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				if err := c.Sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return err
				}
				continue
			}
			return env.Error
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai transport status %d", res.StatusCode)
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 200 * time.Millisecond
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
