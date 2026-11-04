package kvstore

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

const (
	reminderPrefix   = "meet_reminder_"
	reminderIndexKey = "meet_reminder_index"
)

func reminderKey(id string) string {
	return reminderPrefix + id
}

func (kv *Client) StoreScheduledReminder(reminder *ScheduledReminder) error {
	data, err := json.Marshal(reminder)
	if err != nil {
		return errors.Wrap(err, "marshal scheduled reminder")
	}
	if _, err = kv.client.KV.Set(reminderKey(reminder.ID), data); err != nil {
		return errors.Wrap(err, "store scheduled reminder")
	}
	return kv.addToStringList(reminderIndexKey, reminder.ID)
}

func (kv *Client) GetScheduledReminder(id string) (*ScheduledReminder, error) {
	var data []byte
	if err := kv.client.KV.Get(reminderKey(id), &data); err != nil {
		return nil, errors.Wrap(err, "get scheduled reminder")
	}
	if len(data) == 0 {
		return nil, nil
	}
	var reminder ScheduledReminder
	if err := json.Unmarshal(data, &reminder); err != nil {
		return nil, errors.Wrap(err, "decode scheduled reminder")
	}
	return &reminder, nil
}

func (kv *Client) ListScheduledReminderIDs() ([]string, error) {
	var ids []string
	if err := kv.client.KV.Get(reminderIndexKey, &ids); err != nil {
		return nil, errors.Wrap(err, "list scheduled reminders")
	}
	return ids, nil
}

// ClaimScheduledReminder prevents two pollers from posting the same reminder.
// The claim precedes CreatePost; an ambiguous post failure is deliberately not retried.
func (kv *Client) ClaimScheduledReminder(id string, now time.Time) (bool, error) {
	claimed := false
	err := kv.client.KV.SetAtomicWithRetries(reminderKey(id), func(oldValue []byte) (any, error) {
		claimed = false
		if len(oldValue) == 0 {
			return nil, nil
		}
		var reminder ScheduledReminder
		if err := json.Unmarshal(oldValue, &reminder); err != nil {
			return nil, err
		}
		if reminder.Claimed || reminder.StartAt.Add(-time.Duration(reminder.MinutesBefore)*time.Minute).After(now) {
			return reminder, nil
		}
		reminder.Claimed = true
		claimed = true
		return reminder, nil
	})
	if err != nil {
		return false, errors.Wrap(err, "claim scheduled reminder")
	}
	return claimed, nil
}

func (kv *Client) DeleteScheduledReminder(id string) error {
	if err := kv.client.KV.Delete(reminderKey(id)); err != nil {
		return errors.Wrap(err, "delete scheduled reminder")
	}
	return kv.removeFromStringList(reminderIndexKey, id)
}
