// +build ci

package main

// Rename this file to config.go and change the variables,
// also you probably want to change +build ci to +build !ci

import (
	"time"
)

const (
	flowdockAPIKey      = "CHANGE THIS" // The Notifybot api key
	notificationStorage = "/tmp/flowdock_notifications"
	prefix              = "%"
	fastPrefix          = "%%"
	slowPrefix          = "%"
	fasterPrefix        = "%%%"
	fastDelay           = 2 * time.Hour
	fasterDelay         = 25 * time.Minute
)
