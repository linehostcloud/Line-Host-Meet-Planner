package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/linehostcloud/Line-Host-Meet-Planner/server/command"
	"github.com/linehostcloud/Line-Host-Meet-Planner/server/store/kvstore"
)

var googleCalendarURL = "https://www.googleapis.com/calendar/v3"

var errScheduleInput = errors.New("invalid meeting schedule")
var errGoogleNotConnected = errors.New("user not connected to Google")

type scheduleMeetingRequest struct {
	ChannelID       string   `json:"channel_id"`
	Topic           string   `json:"topic"`
	StartAt         string   `json:"start_at"`
	EndAt           string   `json:"end_at"`
	Attendees       []string `json:"attendees"`
	TimeZone        string   `json:"time_zone"`
	ReminderMinutes []int    `json:"reminder_minutes"`
}

type calendarEvent struct {
	ID             string `json:"id"`
	HTMLLink       string `json:"htmlLink"`
	HangoutLink    string `json:"hangoutLink"`
	ConferenceData struct {
		EntryPoints []struct {
			EntryPointType string `json:"entryPointType"`
			URI            string `json:"uri"`
		} `json:"entryPoints"`
	} `json:"conferenceData"`
}

func (e *calendarEvent) meetURL() string {
	if strings.HasPrefix(e.HangoutLink, "https://meet.google.com/") {
		return e.HangoutLink
	}
	for _, point := range e.ConferenceData.EntryPoints {
		if point.EntryPointType == "video" && strings.HasPrefix(point.URI, "https://meet.google.com/") {
			return point.URI
		}
	}
	return ""
}

func validateSchedule(req scheduleMeetingRequest, now time.Time) (time.Time, time.Time, []string, *time.Location, error) {
	if req.ChannelID == "" || strings.TrimSpace(req.Topic) == "" || len([]rune(req.Topic)) > 200 {
		return time.Time{}, time.Time{}, nil, nil, errScheduleInput
	}
	location, err := time.LoadLocation(req.TimeZone)
	if err != nil || req.TimeZone == "" {
		return time.Time{}, time.Time{}, nil, nil, errScheduleInput
	}
	start, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		return time.Time{}, time.Time{}, nil, nil, errScheduleInput
	}
	end, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil || !start.After(now.Add(time.Minute)) || !end.After(start) || end.Sub(start) > 12*time.Hour {
		return time.Time{}, time.Time{}, nil, nil, errScheduleInput
	}
	if len(req.Attendees) > 50 {
		return time.Time{}, time.Time{}, nil, nil, errScheduleInput
	}
	if req.ReminderMinutes != nil {
		if len(req.ReminderMinutes) > 7 {
			return time.Time{}, time.Time{}, nil, nil, errScheduleInput
		}
		allowed := map[int]bool{5: true, 10: true, 15: true, 30: true, 60: true, 120: true, 1440: true}
		seenMinutes := make(map[int]bool)
		for _, minutes := range req.ReminderMinutes {
			if !allowed[minutes] || seenMinutes[minutes] || !start.Add(-time.Duration(minutes)*time.Minute).After(now.Add(time.Minute)) {
				return time.Time{}, time.Time{}, nil, nil, errScheduleInput
			}
			seenMinutes[minutes] = true
		}
	}
	seen := make(map[string]bool)
	attendees := make([]string, 0, len(req.Attendees))
	for _, raw := range req.Attendees {
		address, parseErr := mail.ParseAddress(strings.TrimSpace(raw))
		if parseErr != nil || address.Name != "" || address.Address != strings.TrimSpace(raw) {
			return time.Time{}, time.Time{}, nil, nil, errScheduleInput
		}
		email := strings.ToLower(address.Address)
		if !seen[email] {
			seen[email] = true
			attendees = append(attendees, email)
		}
	}
	return start, end, attendees, location, nil
}

func (p *Plugin) createCalendarEvent(token *kvstore.OAuth2Token, topic string, start, end time.Time, attendees []string) (*calendarEvent, error) {
	requestID, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generate conference request ID: %w", err)
	}
	type eventDateTime struct {
		DateTime string `json:"dateTime"`
	}
	type attendee struct {
		Email string `json:"email"`
	}
	guests := make([]attendee, 0, len(attendees))
	for _, email := range attendees {
		guests = append(guests, attendee{Email: email})
	}
	body, err := json.Marshal(struct {
		Summary        string        `json:"summary"`
		Start          eventDateTime `json:"start"`
		End            eventDateTime `json:"end"`
		Attendees      []attendee    `json:"attendees,omitempty"`
		ConferenceData struct {
			CreateRequest struct {
				RequestID             string `json:"requestId"`
				ConferenceSolutionKey struct {
					Type string `json:"type"`
				} `json:"conferenceSolutionKey"`
			} `json:"createRequest"`
		} `json:"conferenceData"`
	}{
		Summary:   topic,
		Start:     eventDateTime{DateTime: start.Format(time.RFC3339)},
		End:       eventDateTime{DateTime: end.Format(time.RFC3339)},
		Attendees: guests,
		ConferenceData: struct {
			CreateRequest struct {
				RequestID             string `json:"requestId"`
				ConferenceSolutionKey struct {
					Type string `json:"type"`
				} `json:"conferenceSolutionKey"`
			} `json:"createRequest"`
		}{CreateRequest: struct {
			RequestID             string `json:"requestId"`
			ConferenceSolutionKey struct {
				Type string `json:"type"`
			} `json:"conferenceSolutionKey"`
		}{RequestID: requestID, ConferenceSolutionKey: struct {
			Type string `json:"type"`
		}{Type: "hangoutsMeet"}}},
	})
	if err != nil {
		return nil, err
	}
	endpoint := googleCalendarURL + "/calendars/primary/events?conferenceDataVersion=1&sendUpdates=all"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calendar request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, command.ErrNeedsReconnect
	}
	if resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if strings.Contains(string(body), "insufficientPermissions") || strings.Contains(string(body), "ACCESS_TOKEN_SCOPE_INSUFFICIENT") {
			return nil, ErrInsufficientScopes
		}
		return nil, fmt.Errorf("calendar API denied event creation: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("calendar API returned HTTP %d", resp.StatusCode)
	}
	var event calendarEvent
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&event); err != nil {
		return nil, fmt.Errorf("decode calendar event: %w", err)
	}
	if event.ID == "" || event.HTMLLink == "" {
		return nil, errors.New("calendar API omitted event identifier or link")
	}
	return &event, nil
}

func (p *Plugin) fetchCalendarEvent(token *kvstore.OAuth2Token, eventID string) (*calendarEvent, error) {
	endpoint := googleCalendarURL + "/calendars/primary/events/" + url.PathEscape(eventID)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar event lookup returned HTTP %d", resp.StatusCode)
	}
	var event calendarEvent
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (p *Plugin) resolveMeetURL(token *kvstore.OAuth2Token, event *calendarEvent) (string, error) {
	if link := event.meetURL(); link != "" {
		return link, nil
	}
	// Google can create conference details asynchronously.
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		updated, err := p.fetchCalendarEvent(token, event.ID)
		if err != nil {
			return "", err
		}
		if link := updated.meetURL(); link != "" {
			return link, nil
		}
	}
	return "", errors.New("Google has not generated the Meet link yet")
}

// ScheduleMeeting creates one Google Calendar event and posts its Meet link in the current channel.
// If Google has not generated a link, the event remains in Calendar and the caller receives its URL.
func (p *Plugin) ScheduleMeeting(userID string, req scheduleMeetingRequest) (string, string, error) {
	start, end, attendees, location, err := validateSchedule(req, time.Now())
	if err != nil {
		return "", "", err
	}
	if !p.API.HasPermissionToChannel(userID, req.ChannelID, model.PermissionCreatePost) {
		return "", "", ErrNoChannelPermission
	}
	if p.getConfiguration().RestrictMeetingCreation {
		channel, appErr := p.API.GetChannel(req.ChannelID)
		if appErr != nil {
			return "", "", appErr
		}
		if channel.Type == model.ChannelTypeOpen {
			return "", "", command.ErrPublicChannelRestricted
		}
	}
	token, err := p.getValidToken(userID)
	if err != nil {
		return "", "", err
	}
	if token == nil {
		return "", "", errGoogleNotConnected
	}
	event, err := p.createCalendarEvent(token, strings.TrimSpace(req.Topic), start, end, attendees)
	if err != nil {
		return "", "", err
	}
	meetURL, err := p.resolveMeetURL(token, event)
	if err != nil {
		return "", event.HTMLLink, err
	}
	minutesList := req.ReminderMinutes
	if minutesList == nil && start.Add(-5*time.Minute).After(time.Now().Add(time.Minute)) {
		minutesList = []int{5}
	}
	topic := escapeMeetingTopic(req.Topic)
	message := fmt.Sprintf("📅 **%s**\n\nGoogle Meet: %s\nHorário: %s – %s (%s)\n[Ver no Google Calendar](%s)", topic, meetURL, start.In(location).Format("02/01/2006 15:04"), end.In(location).Format("02/01/2006 15:04"), req.TimeZone, event.HTMLLink)
	if len(minutesList) > 0 {
		labels := make([]string, 0, len(minutesList))
		for _, minutes := range minutesList {
			labels = append(labels, reminderLeadLabel(minutes))
		}
		message += "\nAvisos neste canal antes da reunião: " + strings.Join(labels, ", ") + "."
	}
	post := &model.Post{UserId: userID, ChannelId: req.ChannelID, Message: message}
	createdPost, appErr := p.API.CreatePost(post)
	if appErr != nil {
		return "", event.HTMLLink, fmt.Errorf("calendar event created but Mattermost post failed: %w", appErr)
	}
	store := p.getKVStore()
	for _, minutes := range minutesList {
		reminder := &kvstore.ScheduledReminder{
			ID:            fmt.Sprintf("%s_%d", createdPost.Id, minutes),
			ChannelID:     req.ChannelID,
			OrganizerID:   userID,
			Topic:         strings.TrimSpace(req.Topic),
			MeetURL:       meetURL,
			StartAt:       start,
			TimeZone:      req.TimeZone,
			MinutesBefore: minutes,
		}
		if store == nil || store.StoreScheduledReminder(reminder) != nil {
			p.API.LogError("Failed to schedule meeting reminder", "post_id", createdPost.Id, "minutes_before", minutes)
			p.API.SendEphemeralPost(userID, &model.Post{ChannelId: req.ChannelID, Message: "⚠️ A reunião foi agendada, mas não foi possível salvar um dos lembretes no Mattermost. Confira o evento no Google Calendar."})
			break
		}
	}
	return meetURL, event.HTMLLink, nil
}

func escapeMeetingTopic(topic string) string {
	return strings.NewReplacer("*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "`", "\\`", "\n", " ", "\r", " ").Replace(strings.TrimSpace(topic))
}
