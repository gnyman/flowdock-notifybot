package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
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
	MessageID int64
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
		if err != nil {
			log.Printf("Error could not decode %v", dec)
			goto err
		}
		log.Printf("Restored notifications %v", decodedData)
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
	log.Println("Updated notification cache...")
	return nil
}

func addNotification(notifications map[string]map[string]Notification, atTime time.Time, targetUsername string, fromUsername string, threadID string, flowID string, messageID int64) error {
	fmt.Println(notifications)
	if _, exists := usernames[targetUsername]; !exists {
		return fmt.Errorf("We do not know that user")
	}
	notifications[usernames[targetUsername]][threadID] = Notification{atTime, threadID, flowID, fromUsername, messageID}
	return nil
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
		usernames[strings.ToLower(c.Users[userID].Nick)] = userID
		if _, ok := notifications[userID]; !ok {
			notifications[userID] = make(map[string]Notification)
		}
	}
	log.Println(usernames)

	location, err := time.LoadLocation("Europe/Helsinki")
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
						if time.Now().In(location).After(notif.Timestamp) {
							log.Printf("Sending notification due to no activity, %s after %s", notif.Timestamp, time.Now())
							pingUser := c.Users[userID].Nick
							message := fmt.Sprintf("@%v, slow ping from %v from [here](https://www.flowdock.com/app/walkbase/%s/messages/%d)", pingUser, notif.Pinger, flows[notif.Flow].APIName, notif.MessageID)
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
							saveNotifications(notifications, notificationStorage)
						}
					}
				}
			}
		case event := <-events:
			switch event := event.(type) {
			case flowdock.MessageEvent:
				log.Printf("Message event %v", event)
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

				var notifRegex = regexp.MustCompile(fmt.Sprintf(`(\%s+)([\wåäö]+)`, prefix))

				for _, field := range notifRegex.FindAllStringSubmatch(event.Content, -1) {
					if len(field) < 2 {
						continue
					}
					possiblePrefix := field[1]
					possibleUsername := field[2]
					// Check first if the username is a known username, if not skip
					if _, ok := usernames[strings.ToLower(possibleUsername)]; !ok {
						continue
					}
					pinger := c.Users[event.UserID].Nick

					var notificationTime time.Time
					var notifyTag string
					if possiblePrefix == fastPrefix {
						notificationTime = time.Now().In(location).Add(fastDelay)
						notifyTag = fmt.Sprintf("notify-short-%v", possibleUsername)
					}
					if possiblePrefix == slowPrefix {
						notificationTime = NextWorkdayAtNine()
						notifyTag = fmt.Sprintf("notify-long-%v", possibleUsername)
					}
					if possiblePrefix == fasterPrefix {
						notificationTime = time.Now().In(location).Add(fasterDelay)
						notifyTag = fmt.Sprintf("notify-shorter-%v", possibleUsername)
					}
					if !notificationTime.IsZero() {
						log.Printf("%s requested notification for %s at %v", pinger, possibleUsername, notificationTime)
						err = addNotification(notifications, notificationTime, possibleUsername, pinger, event.ThreadID, event.Flow, event.ID)
						if err != nil {
							log.Printf("Error adding notification...")
						}
						flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{notifyTag})
						saveNotifications(notifications, notificationStorage)
					} else {
						log.Println("No time was set for notification")
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

				var notifRegex = regexp.MustCompile(fmt.Sprintf(`(\%s+)([\wåäö]+)`, prefix))

				for _, field := range notifRegex.FindAllStringSubmatch(event.Content.Text, -1) {
					if len(field) < 2 {
						continue
					}
					possiblePrefix := field[1]
					possibleUsername := field[2]
					// Check first if the username is a known username, if not skip
					if _, ok := usernames[strings.ToLower(possibleUsername)]; !ok {
						continue
					}
					pinger := c.Users[event.UserID].Nick

					var notificationTime time.Time
					var notifyTag string
					if possiblePrefix == fastPrefix {
						notificationTime = time.Now().In(location).Add(fastDelay)
						notifyTag = fmt.Sprintf("notify-short-%v", possibleUsername)
					}
					if possiblePrefix == slowPrefix {
						notificationTime = NextWorkdayAtNine()
						notifyTag = fmt.Sprintf("notify-long-%v", possibleUsername)
					}
					if possiblePrefix == fasterPrefix {
						notificationTime = time.Now().In(location).Add(fasterDelay)
						notifyTag = fmt.Sprintf("notify-shorter-%v", possibleUsername)
					}
					if !notificationTime.IsZero() {
						log.Printf("%s requested notification for %s at %v", pinger, possibleUsername, notificationTime)
						err = addNotification(notifications, notificationTime, possibleUsername, pinger, messageID, event.Flow, event.ID)
						if err != nil {
							log.Printf("Error adding notification...")
						}
						flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{notifyTag})
						saveNotifications(notifications, notificationStorage)
					} else {
						log.Println("No time was set for notification")
					}
				}

				//		case flowdock.MessageEditEvent:
				//			log.Printf("Looks like @%s just updated their previous message: '%s'. New message is '%s'", c.DetailsForUser(event.UserID).Nick, messageStore[event.Content.MessageID], event.Content.UpdatedMessage)
			case flowdock.UserActivityEvent:
				log.Printf("User activity event %v", event)
				continue // Especially with > 10 people in your org, you will get MANY of these events.
			case flowdock.ActionEvent:
				log.Printf("Action event %v", event)
				// If we get a flow-change, reload flows and users
				if event.Type == "flow-change" {
					log.Println("Flow-change event, reconnecting and updating stuff...")
					c = flowdock.NewClient(flowdockAPIKey)
					err := c.Connect(nil, events)
					if err != nil {
						log.Printf("Error could not reconnect %v", err)
						time.Sleep(15 * time.Second)
					}
					flows = make(map[string]flowdock.Flow)
					for _, flow := range c.AvailableFlows {
						flows[flow.ID] = flow
					}
					for userID, _ := range c.Users {
						usernames[strings.ToLower(c.Users[userID].Nick)] = userID
						if _, ok := notifications[userID]; !ok {
							notifications[userID] = make(map[string]Notification)
						}
					}
				}
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
