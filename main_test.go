package main

import (
	"testing"
	"time"
)

func TestNextWorkdayNine(t *testing.T) {
	nextWorkDayAtNine := NextWorkdayAtNine()
	hour, min, sec := nextWorkDayAtNine.Clock()
	if hour != 9 && min != 0 && sec != 0 {
		t.Error("Expected NextWorkdayAtNine to be at 9:00:00")
	}
}

func TestStoreAndRestoreNotifications(t *testing.T) {
	notifications := make(map[string]map[string]Notification)

	usernames = make(map[string]string)
	usernames["Björn"] = "12345"
	notifications["12345"] = make(map[string]Notification)

	var time = time.Now()
	var targetUser = "Björn"
	var fromUser = "Gabriel"
	var threadID = "12345abcdefg"
	var flowID = "abcdefg123456"
	var messageID int64 = 12345
	err := addNotification(notifications, time, targetUser, fromUser, threadID, flowID, messageID)

	if err != nil {
		t.Errorf("Could not add notification, error was %s", err)
	}

	path := "/tmp/test-flowdock-notifications.gob"
	saveNotifications(notifications, path)

	restoredNotifications := restoreNotifications(path)

	expectedNotification := Notification{time, threadID, flowID, fromUser, messageID}
	if restoredNotifications["12345"]["12345abcdefg"] != expectedNotification {
		t.Error("Notification not found in map as expected")
	}

	t.Log("%v", notifications)
}

func TestAddNotification(t *testing.T) {
	notifications := make(map[string]map[string]Notification)

	usernames = make(map[string]string)
	usernames["Björn"] = "12345"
	notifications["12345"] = make(map[string]Notification)

	var time = time.Now()
	var targetUser = "Björn"
	var fromUser = "Gabriel"
	var threadID = "12345abcdefg"
	var flowID = "abcdefg123456"
	var messageID int64 = 1234
	addNotification(notifications, time, targetUser, fromUser, threadID, flowID, messageID)

	expectedNotification := Notification{time, threadID, flowID, fromUser, messageID}
	if notifications["12345"]["12345abcdefg"] != expectedNotification {
		t.Error("Notification not found in map as expected")
	}

	unexpectedNotification := Notification{time, "wrongThreadId", flowID, fromUser, messageID}
	if notifications["12345"]["12345abcdefg"] == unexpectedNotification {
		t.Error("Matches even though it should not")
	}

	err := addNotification(notifications, time, "nonExistingTargetUser", fromUser, threadID, flowID, messageID)

	if err == nil {
		t.Error("Expected error when adding non existing user but got none")
	}
}

/*func TestParseStringForSlowNotificationRequest(t *testing.T) {
	stringWithSlowNotification := "!Gabriel lolwut"

	notifications := make(map[string]map[string]Notification)

	location, _ = time.LoadLocation("Europe/Zurich")

	usernames = make(map[string]string)
	usernames["gabriel"] = "12345"
	notifications["12345"] = make(map[string]Notification)

	orgID := "walkbase"
	threadID := "wrongThreadId"
	flowID := "asdfg"
	messageID := "poiuyt"
	pinger := "Björn"
	time := time.Now()

	parseStringForNotificationRequests(stringWithSlowNotification, orgID, threadID, flowID, messageID, pinger)

	expectedNotification := Notification{time, "wrongThreadId", flowID, pinger}
	if notifications["12345"]["12345abcdefg"] != expectedNotification {
		t.Errorf("Does not match\n%v\n%v", expectedNotification, notifications["12345"]["12345abcdefg"])
	}

}*/
