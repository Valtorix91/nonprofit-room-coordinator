package matchmaking

import (
	"errors"
	"sort"
	"strings"
)

type Player struct {
	ID          string `json:"id"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name"`
}

type Request struct {
	CampaignID     string   `json:"campaign_id"`
	Players        []Player `json:"players"`
	ReceiptIDs     []string `json:"receipt_ids"`
	ReminderIDs    []string `json:"reminder_ids"`
	ReportPeriod   string   `json:"report_period"`
	IdempotencyKey string   `json:"idempotency_key"`
}

type RoomState struct {
	CampaignID       string   `json:"campaign_id"`
	Channel          string   `json:"channel"`
	PlayerIDs        []string `json:"player_ids"`
	ReceiptCount     int      `json:"receipt_count"`
	PendingReminders int      `json:"pending_reminders"`
	ReportPeriod     string   `json:"report_period"`
}

var ErrNoMatch = errors.New("a match needs at least one donor and one volunteer")

func Match(req Request) (RoomState, error) {
	if strings.TrimSpace(req.CampaignID) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return RoomState{}, errors.New("campaign_id and idempotency_key are required")
	}

	roles := map[string]int{}
	ids := make([]string, 0, len(req.Players))
	for _, player := range req.Players {
		roles[player.Role]++
		ids = append(ids, player.ID)
	}
	if roles["donor"] == 0 || roles["volunteer"] == 0 {
		return RoomState{}, ErrNoMatch
	}
	sort.Strings(ids)

	return RoomState{
		CampaignID:       req.CampaignID,
		Channel:          "campaign-" + req.CampaignID,
		PlayerIDs:        ids,
		ReceiptCount:     len(req.ReceiptIDs),
		PendingReminders: len(req.ReminderIDs),
		ReportPeriod:     req.ReportPeriod,
	}, nil
}
