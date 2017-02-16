package main

import (
	"reflect"
	"testing"
	"time"
)

func TestNotificationsAdd(t *testing.T) {
	notifications := NewNotifications()

	if len(notifications) != 0 {
		t.Errorf("Add: len(notifcations) is not empty")
	}

	notification := NewNotification(time.Now(), "pinger", "threadID", "flowID", 0)
	notifications.Add(notification, "user1", "thread1")
	notifications.Add(notification, "user1", "thread2")
	notifications.Add(notification, "user2", "thread3")

	if len(notifications["user1"]) != 2 && len(notifications["user2"]) != 1 {
		t.Errorf("Add: notifications are missing")
	}

	if notifications["user1"]["thread1"] != notification {
		t.Errorf("Notification not found in map as expected")
	}
}

func TestNotificationsDelete(t *testing.T) {
	notifications := NewNotifications()

	notification := NewNotification(time.Now(), "pinger", "threadID", "flowID", 0)
	notifications.Add(notification, "user1", "thread1")
	notifications.Add(notification, "user1", "thread2")
	notifications.Add(notification, "user2", "thread3")

	notifications.Delete("user1", "thread2")

	if len(notifications["user1"]) != 1 && len(notifications["user2"]) != 1 {
		t.Errorf("Add: notifications are missing")
	}
}

func TestNotificationsStoreAndRestore(t *testing.T) {
	notifications := NewNotifications()

	notification := NewNotification(time.Now(), "pinger", "threadID", "flowID", 0)
	notifications.Add(notification, "user1", "thread1")
	notifications.Add(notification, "user1", "thread2")
	notifications.Add(notification, "user2", "thread3")

	file := "/tmp/test-flowdock-notifications.gob"
	err := notifications.Save(file)
	if err != nil {
		t.Fatal(err)
	}

	restoredNotifications := NewNotifications()
	restored, err := restoredNotifications.Restore(file)
	if err != nil {
		t.Fatal(err)
	}

	if restored != 3 {
		t.Errorf("Restore: wanted %d, got %d", 3, restored)
	}
	if !reflect.DeepEqual(notifications, restoredNotifications) {
		t.Errorf("wanted %+v", notifications)
		t.Errorf("got %+v", restoredNotifications)
	}
}
