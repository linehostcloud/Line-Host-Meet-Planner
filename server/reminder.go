package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/linehostcloud/Line-Host-Meet-Planner/server/store/kvstore"
)

const reminderPollInterval = 30 * time.Second
const reminderLateGrace = 2 * time.Minute

func (p *Plugin) startReminderPoller() {
	stop := make(chan struct{})
	p.reminderStop = stop
	go func() {
		p.runReminderCycle(time.Now())
		ticker := time.NewTicker(reminderPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case now := <-ticker.C:
				p.runReminderCycle(now)
			}
		}
	}()
}

func (p *Plugin) stopReminderPoller() {
	if p.reminderStop != nil {
		close(p.reminderStop)
		p.reminderStop = nil
	}
}

func (p *Plugin) runReminderCycle(now time.Time) {
	store := p.getKVStore()
	if store == nil {
		return
	}
	ids, err := store.ListScheduledReminderIDs()
	if err != nil {
		p.API.LogError("Failed to list scheduled reminders", "error", err.Error())
		return
	}
	for _, id := range ids {
		p.processScheduledReminder(store, id, now)
	}
}

func (p *Plugin) processScheduledReminder(store kvstore.KVStore, id string, now time.Time) {
	reminder, err := store.GetScheduledReminder(id)
	if err != nil {
		p.API.LogError("Failed to load scheduled reminder", "reminder_id", id, "error", err.Error())
		return
	}
	if reminder == nil {
		p.deleteScheduledReminder(store, id)
		return
	}
	due := reminder.StartAt.Add(-time.Duration(reminder.MinutesBefore) * time.Minute)
	if reminder.Claimed || !now.Before(reminder.StartAt) || now.Sub(due) > reminderLateGrace {
		p.deleteScheduledReminder(store, id)
		return
	}
	if now.Before(due) {
		return
	}
	claimed, err := store.ClaimScheduledReminder(id, now)
	if err != nil {
		p.API.LogError("Failed to claim scheduled reminder", "reminder_id", id, "error", err.Error())
		return
	}
	if !claimed {
		return
	}

	message := p.scheduledReminderMessage(reminder)
	if _, appErr := p.API.CreatePost(&model.Post{UserId: p.botID, ChannelId: reminder.ChannelID, Message: message}); appErr != nil {
		p.API.LogError("Failed to post scheduled reminder", "reminder_id", id, "error", appErr.Error())
	}
	p.deleteScheduledReminder(store, id)
}

func (p *Plugin) deleteScheduledReminder(store kvstore.KVStore, id string) {
	if err := store.DeleteScheduledReminder(id); err != nil {
		p.API.LogError("Failed to clean up scheduled reminder", "reminder_id", id, "error", err.Error())
	}
}

func (p *Plugin) scheduledReminderMessage(reminder *kvstore.ScheduledReminder) string {
	label := reminderLeadLabel(reminder.MinutesBefore)
	location, err := time.LoadLocation(reminder.TimeZone)
	if err != nil {
		location = time.UTC
	}
	message := fmt.Sprintf("⏰ **Lembrete: %s antes da reunião**\n\n**%s**\nInício: %s (%s)\nGoogle Meet: %s",
		label, escapeMeetingTopic(reminder.Topic), reminder.StartAt.In(location).Format("02/01/2006 15:04"), reminder.TimeZone, reminder.MeetURL)
	user, appErr := p.API.GetUser(reminder.OrganizerID)
	if appErr == nil && user != nil && user.Username != "" {
		message += "\n\n@" + user.Username
	}
	return message
}

func reminderLeadLabel(minutes int) string {
	switch minutes {
	case 1440:
		return "1 dia"
	case 120:
		return "2 horas"
	case 60:
		return "1 hora"
	case 30, 15, 10, 5:
		return fmt.Sprintf("%d minutos", minutes)
	default:
		return fmt.Sprintf("%d minutos", minutes)
	}
}
