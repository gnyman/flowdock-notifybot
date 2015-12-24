package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
    
	"github.com/gnyman/flowdock"
)

type ThreadID int64
type Username string

type Notification struct {
	Timestamp time.Time
	Thread    string
	Flow      string
	Pinger    string
}

// Global variables
var notifications map[string]map[string]Notification
var usernames map[string]string

// Return the next workday (not saturday or sunday) at 9 helsinki time
func NextWorkdayAtNine() time.Time {
	location, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		log.Panic("Could not load timezone info")
	}
	now := time.Now().Round(time.Hour).In(location)
	for {
		if now.Weekday() == time.Saturday ||
			now.Weekday() == time.Sunday ||
			now.Hour() != 9 {
			now = now.Add(1 * time.Hour)
		} else {
			break
		}
	}
	return now
}

func restoreNotifications(path string) map[string]map[string]Notification {
	if _, err := os.Stat(path); err == nil {
		rawData, err := ioutil.ReadFile(path)
		if err != nil {
			log.Println("Error could not restore notifications because could not read file :-(")
			goto err
		}
		buffer := bytes.NewBuffer(rawData)
		dec := gob.NewDecoder(buffer)

		var decodedData map[string]map[string]Notification
		err = dec.Decode(&decodedData)
		return decodedData
	}
	log.Println("No notification storage found, not restoring anything")
	err:
		return make(map[string]map[string]Notification)
}

func saveNotifications(notifs map[string]map[string]Notification, path string) error {
	var rawData bytes.Buffer
	enc := gob.NewEncoder(&rawData)
	err := enc.Encode(notifs)
	if err != nil {
		log.Println("Error could not save the notifications")
	}

	ioutil.WriteFile(path, rawData.Bytes(), 0600)
	return nil
}

func addNotification(atTime time.Time, targetUsername string, fromUsername string, threadID string, flowID string) {
    fmt.Println(notifications)
	notifications[usernames[targetUsername]][threadID] = Notification{atTime, threadID, flowID, fromUsername}
}

func main() {
	notifications = restoreNotifications(notificationStorage)

	events := make(chan flowdock.Event)
	c := flowdock.NewClient(flowdockAPIKey)
	err := c.Connect(nil, events)
	if err != nil {
		panic(err)
	}

	usernames = make(map[string]string)

	for userID, _ := range c.Users {
		if _, ok := usernames[userID]; !ok {
			notifications[userID] = make(map[string]Notification)
			usernames[strings.ToLower(c.Users[userID].Nick)] = userID	
		}
	}
	log.Println(usernames)

	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		log.Panic("Could not load timeszone info")
	}

	flows := make(map[string]flowdock.Flow)
	for _, flow := range c.AvailableFlows {
		flows[flow.ID] = flow
	}

	//go ticker(&notifications)
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			//log.Println("Tick, notifications contains %v", notifications)
			for userID, notifs := range notifications {
				if len(notifs) != 0 {
					log.Printf("UserID: %v has notifications %v\n", userID, notifs)
					for threadID, notif := range notifs {
						if time.Now().After(notif.Timestamp) {
							log.Printf("Sending notification due to no activity, %s after %s", notif.Timestamp, time.Now())
							pingUser := c.Users[userID].Nick
							message := fmt.Sprintf("@%v, slow ping from %v", pingUser, notif.Pinger)
							var body []byte
							var err error
							if notif.Thread != "" {
								body, err = flowdock.SendMessageToFlowWithApiKey(flowdockAPIKey, notif.Flow, notif.Thread, message)
							}
							if err != nil {
								log.Panic(err)
							}
							log.Printf("%v\n", string(body))
							delete(notifications[userID], threadID)
						}
					}
				}
			}
		case event := <-events:
			switch event := event.(type) {
			case flowdock.MessageEvent:
                log.Println("Message event")
				orgNflow := strings.Split(event.Flow, ":")
				var org string
				var flow string
				if len(orgNflow) != 2 {
					if _, ok := flows[event.Flow]; ok {
						org = flows[event.Flow].Organization.APIName
						flow = flows[event.Flow].APIName
					} else {
						log.Printf("Odd, we got a message from a flow we do not know, maybe we joined a new channel, reconnecting")
						break
					}
				} else {
					org = orgNflow[0]
					flow = orgNflow[1]
				}

				if _, found := notifications[event.UserID][event.ThreadID]; found {
					log.Printf("User %v was active in thread %v for which he had a notificating pending, clearing notification", event.UserID, event.ThreadID)
					delete(notifications[event.UserID], event.ThreadID)
					nickClear := fmt.Sprintf("cleared-%s", c.Users[event.UserID].Nick)
					flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{nickClear})
				}

				if strings.HasPrefix(event.Content, "!help") {
					flowdock.SendMessageToFlowWithApiKey(flowdockAPIKey, event.Flow, event.ThreadID, "Notifybot does slow notifications, create a slow notification for a person by doing !<nick>  or !!<nick>. The first will @<nick> the person the following day at 09:00 Finnish time, the second will notify him one hour later. If the target is active in the thread, both kind of notifications will be cleared.")
				}

				for _, field := range strings.Fields(event.Content) {
					if strings.HasPrefix(field, strings.Repeat(prefix,2)) {
						possibleUsername := strings.TrimRight(strings.ToLower(strings.TrimLeft(field, prefix)), ",.? ")
						log.Printf("PossibleUsername: %v", possibleUsername)
						log.Printf("Usernames: %v", usernames)
						if _, ok := usernames[possibleUsername]; ok {
							log.Printf("Found username, adding to notifications %s!\n", possibleUsername)
							pinger := c.Users[event.UserID].Nick
							inOneHour := time.Now().In(location).Add(5 * time.Second)
                            addNotification(inOneHour, possibleUsername, pinger, event.ThreadID, event.Flow)
							//notifications[usernames[possibleUsername]][event.ThreadID] = Notification{inOneHour, event.ThreadID, event.Flow, pinger}
							notifyTag := fmt.Sprintf("notify-short-%v", possibleUsername)
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{notifyTag})
						}
						log.Printf("%s", notifications[event.UserID][event.ThreadID])
					}
					if strings.HasPrefix(field, prefix) && !strings.HasPrefix(field, strings.Repeat(prefix,2)) {
						log.Printf("Saw notification thingie, %s", field)
						possibleUsername := strings.TrimRight(strings.ToLower(strings.TrimLeft(field, prefix)), ",.? ")
						if _, ok := usernames[possibleUsername]; ok {
							log.Printf("Found username, adding to notifications %s!\n", possibleUsername)
							pinger := c.Users[event.UserID].Nick
							nextWorkDayAtNine := NextWorkdayAtNine()
							notifications[usernames[possibleUsername]][event.ThreadID] = Notification{nextWorkDayAtNine, event.ThreadID, event.Flow, pinger}
							notifyTag := fmt.Sprintf("notify-long-%v", possibleUsername)
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{notifyTag})
							saveNotifications(notifications, notificationStorage)
						}
						log.Printf("%s", notifications[event.UserID][event.ThreadID])
					}
				}
				log.Printf("%s said (%s): '%s'", c.DetailsForUser(event.UserID).Nick, event.Flow, event.Content)
			case flowdock.CommentEvent:
                log.Println("Comment event")
				orgNflow := strings.Split(event.Flow, ":")
				var org string
				var flow string
				if len(orgNflow) != 2 {
					if _, ok := flows[event.Flow]; ok {
						org = flows[event.Flow].Organization.APIName
						flow = flows[event.Flow].APIName
					} else {
						log.Printf("Odd, we got a message from a flow we do not know, maybe we joined a new channel, reconnecting")
						break
					}
				} else {
					org = orgNflow[0]
					flow = orgNflow[1]
				}

				log.Printf("%s commented (%s): '%s'", c.DetailsForUser(event.UserID).Nick, event.Flow, event.Content.Text)

				var messageID string

				for _, tag := range event.Tags {
					if strings.HasPrefix(tag, "influx:") {
						messageID = strings.TrimPrefix(tag, "influx:")
					}
				}

				if _, found := notifications[event.UserID][messageID]; found {
					log.Printf("User %v was active in comment thread %v for which he had a notificating pending, clearing notification", event.UserID, messageID)
					delete(notifications[event.UserID], messageID)
					nickClear := fmt.Sprintf("cleared-%s", c.Users[event.UserID].Nick)
					flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{nickClear})
				}

				if strings.HasPrefix(event.Content.Text, "!help") {
					flowdock.SendCommentToFlowWithApiKey(flowdockAPIKey, event.Flow, messageID, "Notifybot does slow notifications, create a slow notification for a person by doing !<nick>  or !!<nick>. The first will @<nick> the person the following day at 09:00 Finnish time, the second will notify him one hour later. If the target is active in the thread, both kind of notifications will be cleared.")
				}
				for _, field := range strings.Fields(event.Content.Text) {
					if strings.HasPrefix(field, strings.Repeat(prefix,2)) {
						possibleUsername := strings.TrimRight(strings.ToLower(strings.TrimLeft(field, prefix)), ",.? ")
						if _, ok := usernames[possibleUsername]; ok {
							log.Printf("Found username, adding to notifications %s!\n", possibleUsername)
							pinger := c.Users[event.UserID].Nick
							inOneHour := time.Now().In(location).Add(5 * time.Second)
							notifications[usernames[possibleUsername]][messageID] = Notification{inOneHour, messageID, event.Flow, pinger}
							notifyTag := fmt.Sprintf("notify-short-%v", possibleUsername)
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{notifyTag})
						}
						log.Printf("%s", notifications[event.UserID][messageID])
						break
					}
					if strings.HasPrefix(field, prefix) && !strings.HasPrefix(field, strings.Repeat(prefix,2)) {
						log.Printf("Saw notification thingie, %s", field)
						possibleUsername := strings.TrimRight(strings.ToLower(strings.TrimLeft(field, prefix)), ",.? ")
						if _, ok := usernames[possibleUsername]; ok {
							log.Printf("Found username, adding to notifications %s!\n", possibleUsername)
							pinger := c.Users[event.UserID].Nick
							nextWorkDayAtNine := NextWorkdayAtNine()
							//addNotification(nextWorkDayAtNine, possibleUsername, pinger, messageID)
							notifications[usernames[possibleUsername]][messageID] = Notification{nextWorkDayAtNine, messageID, event.Flow, pinger}
							notifyTag := fmt.Sprintf("notify-long-%v", possibleUsername)
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{notifyTag})
							saveNotifications(notifications, notificationStorage)
						}
						log.Printf("%s", notifications[event.UserID][messageID])
						break
					}
				}
				//		case flowdock.MessageEditEvent:
				//			log.Printf("Looks like @%s just updated their previous message: '%s'. New message is '%s'", c.DetailsForUser(event.UserID).Nick, messageStore[event.Content.MessageID], event.Content.UpdatedMessage)
			case flowdock.UserActivityEvent:
				continue // Especially with > 10 people in your org, you will get MANY of these events.
			case nil:
				c = flowdock.NewClient(flowdockAPIKey)
				err := c.Connect(nil, events)
				if err != nil {
					log.Printf("Error could not recoonect %v", err)
					time.Sleep(15 * time.Second)
				}
			default:
				log.Printf("New event of type %T", event)
			}
		}
	}
}
