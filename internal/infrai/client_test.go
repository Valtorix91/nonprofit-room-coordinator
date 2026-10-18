package infrai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateChannelBoundary(t *testing.T) {
	tests := []struct {
		name         string
		responses    []string
		statuses     []int
		wantCode     string
		wantRequests int
	}{
		{
			name:         "business rejection retains code",
			responses:    []string{`{"ok":false,"data":null,"error":{"code":"INVALID_INPUT","message":"channel rejected"},"metadata":null}`},
			statuses:     []int{http.StatusBadRequest},
			wantCode:     "INVALID_INPUT",
			wantRequests: 1,
		},
		{
			name: "rate limit retries the same write",
			responses: []string{
				`{"ok":false,"data":null,"error":{"code":"RATE_LIMITED","message":"retry later"},"metadata":null}`,
				`{"ok":true,"data":{"channel":"campaign-food-drive"},"error":null,"metadata":null}`,
			},
			statuses:     []int{http.StatusTooManyRequests, http.StatusOK},
			wantRequests: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			var firstKey string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v1/realtime/channel/create" {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("authorization header missing")
				}
				key := r.Header.Get("Idempotency-Key")
				if requests == 0 {
					firstKey = key
				} else if key != firstKey {
					t.Errorf("idempotency key changed across retry")
				}
				w.WriteHeader(tt.statuses[requests])
				_, _ = w.Write([]byte(tt.responses[requests]))
				requests++
			}))
			defer server.Close()

			client := New("test-key")
			client.BaseURL = server.URL
			client.Sleep = func(context.Context, time.Duration) error { return nil }
			err := client.CreateChannel(context.Background(), "campaign-food-drive", "match-17:channel")
			if tt.wantCode == "" && err != nil {
				t.Fatalf("CreateChannel() error = %v", err)
			}
			if tt.wantCode != "" {
				apiErr, ok := err.(*APIError)
				if !ok || apiErr.Code != tt.wantCode {
					t.Fatalf("CreateChannel() error = %#v, want code %s", err, tt.wantCode)
				}
			}
			if requests != tt.wantRequests {
				t.Fatalf("requests = %d, want %d", requests, tt.wantRequests)
			}
		})
	}
}
