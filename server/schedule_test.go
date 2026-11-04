package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/linehostcloud/Line-Host-Meet-Planner/server/store/kvstore"
)

func TestValidateSchedule(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	req := scheduleMeetingRequest{
		ChannelID: "channel", Topic: "Equipe", StartAt: now.Add(time.Hour).Format(time.RFC3339),
		EndAt: now.Add(2 * time.Hour).Format(time.RFC3339), TimeZone: "America/Sao_Paulo",
		Attendees: []string{"A@example.com", "a@example.com", "b@example.com"},
	}
	_, _, attendees, location, err := validateSchedule(req, now)
	if err != nil || location.String() != req.TimeZone || len(attendees) != 2 {
		t.Fatalf("valid schedule: attendees=%v location=%v err=%v", attendees, location, err)
	}
	req.EndAt = now.Format(time.RFC3339)
	if _, _, _, _, err = validateSchedule(req, now); !errors.Is(err, errScheduleInput) {
		t.Fatalf("expected invalid schedule, got %v", err)
	}
	req.EndAt = now.Add(2 * time.Hour).Format(time.RFC3339)
	req.TimeZone = "invalid/zone"
	if _, _, _, _, err = validateSchedule(req, now); !errors.Is(err, errScheduleInput) {
		t.Fatalf("expected invalid time zone, got %v", err)
	}
	req.TimeZone = "America/Sao_Paulo"
	req.ReminderMinutes = []int{1440, 120, 60, 30, 15, 10, 5}
	req.StartAt = now.Add(48 * time.Hour).Format(time.RFC3339)
	req.EndAt = now.Add(49 * time.Hour).Format(time.RFC3339)
	_, _, _, _, err = validateSchedule(req, now)
	require.NoError(t, err)
	req.ReminderMinutes = []int{60, 60}
	_, _, _, _, err = validateSchedule(req, now)
	require.ErrorIs(t, err, errScheduleInput)
	req.ReminderMinutes = []int{1440}
	req.StartAt = now.Add(20 * time.Hour).Format(time.RFC3339)
	req.EndAt = now.Add(21 * time.Hour).Format(time.RFC3339)
	_, _, _, _, err = validateSchedule(req, now)
	require.ErrorIs(t, err, errScheduleInput)
}

func TestScheduleMeetingStoresSelectedReminders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"event1","htmlLink":"https://calendar.google.com/event?eid=event1","hangoutLink":"https://meet.google.com/abc-defg-hij"}`))
	}))
	defer server.Close()
	previous := googleCalendarURL
	googleCalendarURL = server.URL
	defer func() { googleCalendarURL = previous }()

	p, api, kv := setupPlugin(t)
	kv.tokens["organizer"] = &kvstore.OAuth2Token{ClientID: "test-client-id", AccessToken: "test-token", Expiry: time.Now().Add(time.Hour)}
	start := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	_, _, err := p.ScheduleMeeting("organizer", scheduleMeetingRequest{
		ChannelID: "channel", Topic: "Planejamento", StartAt: start.Format(time.RFC3339),
		EndAt: start.Add(time.Hour).Format(time.RFC3339), TimeZone: "America/Sao_Paulo",
		ReminderMinutes: []int{1440, 120, 30},
	})
	require.NoError(t, err)
	require.Len(t, kv.reminders, 3)
	for _, minutes := range []int{1440, 120, 30} {
		reminder := kv.reminders[api.post.Id+"_"+fmt.Sprint(minutes)]
		require.NotNil(t, reminder)
		require.Equal(t, minutes, reminder.MinutesBefore)
	}
}

func TestCreateCalendarEventRequestsMeetAndInvites(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/calendars/primary/events" || r.URL.Query().Get("conferenceDataVersion") != "1" || r.URL.Query().Get("sendUpdates") != "all" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing bearer token")
		}
		var body struct {
			Summary   string `json:"summary"`
			Attendees []struct {
				Email string `json:"email"`
			} `json:"attendees"`
			ConferenceData struct {
				CreateRequest struct {
					ConferenceSolutionKey struct {
						Type string `json:"type"`
					} `json:"conferenceSolutionKey"`
				} `json:"createRequest"`
			} `json:"conferenceData"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Summary != "Equipe" || len(body.Attendees) != 1 || body.Attendees[0].Email != "a@example.com" || body.ConferenceData.CreateRequest.ConferenceSolutionKey.Type != "hangoutsMeet" {
			t.Errorf("invalid Calendar payload: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"event1","htmlLink":"https://calendar.google.com/event?eid=event1","hangoutLink":"https://meet.google.com/abc-defg-hij"}`))
	}))
	defer server.Close()
	previous := googleCalendarURL
	googleCalendarURL = server.URL
	defer func() { googleCalendarURL = previous }()
	p := &Plugin{}
	event, err := p.createCalendarEvent(&kvstore.OAuth2Token{ClientID: "test-client-id", AccessToken: "test-token"}, "Equipe", time.Now().Add(time.Hour), time.Now().Add(2*time.Hour), []string{"a@example.com"})
	if err != nil || event.meetURL() != "https://meet.google.com/abc-defg-hij" {
		t.Fatalf("event=%+v err=%v", event, err)
	}
	if !strings.HasPrefix(event.HTMLLink, "https://calendar.google.com/") {
		t.Fatal("missing calendar link")
	}
}
