package main

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"

	"github.com/linehostcloud/Line-Host-Meet-Planner/server/store/kvstore"
)

func TestScheduledReminderPostsOnceAtSelectedTime(t *testing.T) {
	p, api, kv := setupPlugin(t)
	p.botID = "bot-id"
	api.user = &model.User{Username: "organizer"}
	api.captureAllPosts = true
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	reminder := &kvstore.ScheduledReminder{
		ID: "post-1_120", ChannelID: "channel", OrganizerID: "organizer-id",
		Topic: "Equipe", MeetURL: "https://meet.google.com/abc-defg-hij",
		StartAt: now.Add(2 * time.Hour), TimeZone: "America/Sao_Paulo", MinutesBefore: 120,
	}
	require.NoError(t, kv.StoreScheduledReminder(reminder))

	p.runReminderCycle(now.Add(-time.Minute))
	require.Empty(t, api.allPosts)
	p.runReminderCycle(now)
	require.Len(t, api.allPosts, 1)
	require.Equal(t, "bot-id", api.allPosts[0].UserId)
	require.Contains(t, api.allPosts[0].Message, "2 horas antes")
	require.Contains(t, api.allPosts[0].Message, "@organizer")
	p.runReminderCycle(now.Add(time.Minute))
	require.Len(t, api.allPosts, 1)
	require.Empty(t, kv.reminderIndex)
}

func TestScheduledReminderSkipsExpiredWindow(t *testing.T) {
	p, api, kv := setupPlugin(t)
	api.captureAllPosts = true
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	require.NoError(t, kv.StoreScheduledReminder(&kvstore.ScheduledReminder{
		ID: "post-2_1440", ChannelID: "channel", StartAt: now.Add(time.Hour), MinutesBefore: 1440,
	}))
	p.runReminderCycle(now)
	require.Empty(t, api.allPosts)
	require.Empty(t, kv.reminderIndex)
}
