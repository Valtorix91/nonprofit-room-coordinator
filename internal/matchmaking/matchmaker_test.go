package matchmaking

import (
	"errors"
	"reflect"
	"testing"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		name      string
		players   []Player
		wantIDs   []string
		wantError error
	}{
		{
			name: "donor and volunteer form a room",
			players: []Player{
				{ID: "vol-2", Role: "volunteer", DisplayName: "Rae"},
				{ID: "donor-1", Role: "donor", DisplayName: "Kai"},
			},
			wantIDs: []string{"donor-1", "vol-2"},
		},
		{
			name:      "donors wait for a volunteer",
			players:   []Player{{ID: "donor-1", Role: "donor", DisplayName: "Kai"}},
			wantError: ErrNoMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Match(Request{
				CampaignID:     "food-drive",
				Players:        tt.players,
				ReceiptIDs:     []string{"receipt-8"},
				ReminderIDs:    []string{"reminder-3", "reminder-4"},
				ReportPeriod:   "2026-Q3",
				IdempotencyKey: "match-food-drive-17",
			})
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("Match() error = %v, want %v", err, tt.wantError)
			}
			if tt.wantError != nil {
				return
			}
			if !reflect.DeepEqual(got.PlayerIDs, tt.wantIDs) {
				t.Fatalf("PlayerIDs = %v, want %v", got.PlayerIDs, tt.wantIDs)
			}
			if got.ReceiptCount != 1 || got.PendingReminders != 2 {
				t.Fatalf("counts = receipts:%d reminders:%d", got.ReceiptCount, got.PendingReminders)
			}
		})
	}
}
