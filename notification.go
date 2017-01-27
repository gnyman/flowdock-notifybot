package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

// Notification holds information about a notification
type Notification struct {
	Timestamp time.Time
	Thread    string
	Flow      string
	Pinger    string
	MessageID int64
}

// NewNotification creates a new notification from the given parameters
func NewNotification(t time.Time, pinger, threadID, flowID string, messageID int64) Notification {
	return Notification{t, threadID, flowID, pinger, messageID}
}

// Notifications is a map of Notifications by user and thread ID
type Notifications map[string]map[string]Notification

// NewNotifications returns a empty notifications map
func NewNotifications() Notifications {
	return make(map[string]map[string]Notification)
}

// Restore restores saved notifications from file
func (n Notifications) Restore(file string) (int, error) {
	if _, err := os.Stat(file); err == nil {
		rawData, err := ioutil.ReadFile(file)
		if err != nil {
			return 0, fmt.Errorf("Error could not restore notifications because could not read file :-(")
		}
		buffer := bytes.NewBuffer(rawData)
		dec := gob.NewDecoder(buffer)

		err = dec.Decode(&n)
		if err != nil {
			return 0, fmt.Errorf("Error could not decode %v", dec)
		}
		total := 0
		for _, user := range n {
			total += len(user)
		}
		return total, nil
	}
	return 0, nil
}

// Save saves notifications to file
func (n Notifications) Save(file string) error {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(n)
	if err != nil {
		return fmt.Errorf("Error could not save the notifications")
	}

	ioutil.WriteFile(file, buffer.Bytes(), 0600)
	return nil
}

// Add adds a notification to the map
func (n Notifications) Add(nn Notification, to, threadID string) {
	if _, exists := n[to]; !exists {
		n[to] = make(map[string]Notification)
	}
	n[to][threadID] = nn
}

// Delete deletes a notification from the map
func (n Notifications) Delete(to, threadID string) {
	delete(n[to], threadID)
}
