package main

import (
	"testing"
)

func TestNextWorkdayNine(t *testing.T) {
	nextWorkDayAtNine := NextWorkdayAtNine()
	hour, min, sec := nextWorkDayAtNine.Clock()
	if hour != 9 && min != 0 && sec != 0 {
		t.Error("Expected NextWorkdayAtNine to be at 9:00:00")
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
