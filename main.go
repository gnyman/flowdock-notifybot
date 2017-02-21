package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gnyman/flowdock"

	yaml "gopkg.in/yaml.v2"
)

type ThreadID int64
type Username string

type config struct {
	FlowdockAPIKey string `yaml:"flowdock_api_key"`
	StoragePath    string `yaml:"storage_path"`
	PingPrefix     rune   `yaml:"ping_prefix"`
	RolePrefix     rune   `yaml:"role_prefix"`
}

const (
	fastDelay   = 1 * time.Hour
	fasterDelay = 25 * time.Minute
)

// Global variables
var flowdockAPIKey = ""
var notificationStorage = "/tmp/flowdock_notifications"
var roleStorage = "/tmp/flowdock_roles"
var pingPrefix = "!"
var slowPrefix = "!"
var fastPrefix = "!!"
var fasterPrefix = "!!!"
var rolePrefix = "&"
var notifications Notifications
var users Users
var roles Roles

// Return the next workday (not saturday or sunday) at 9 helsinki time
func NextWorkdayAtNine() time.Time {
	location, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		log.Panic("Could not load timezone info")
	}
	now := time.Now().In(location).Truncate(time.Hour)
	hoursFromNine := time.Duration(9 - now.Hour())
	if hoursFromNine > 0 {
		now = now.Add(hoursFromNine * time.Hour)
	} else {
		now = now.Add((24 + hoursFromNine) * time.Hour)
	}
	switch now.Weekday() {
	case time.Saturday:
		now = now.AddDate(0, 0, 2)
	case time.Sunday:
		now = now.AddDate(0, 0, 1)
	}
	return now
}

// createNotifyTimeAndTag returns the time when the notification shall be sent
// and the tag used
func createNotifyTimeAndTag(prefix, username string, location *time.Location) (time.Time, string) {
	var t time.Time
	var tag string

	if prefix == fastPrefix {
		t = time.Now().In(location).Add(fastDelay)
		tag = fmt.Sprintf("notify-short-%v", username)
	}
	if prefix == slowPrefix {
		t = NextWorkdayAtNine()
		tag = fmt.Sprintf("notify-long-%v", username)
	}
	if prefix == fasterPrefix {
		t = time.Now().In(location).Add(fasterDelay)
		tag = fmt.Sprintf("notify-shorter-%v", username)
	}

	return t, tag
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "config.yaml", "Config file to read settings from")
	flag.Parse()

	// Read settings from config file
	var conf config
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln("Failed to open config file:", err)
	}
	err = yaml.Unmarshal(content, &conf)
	if err != nil {
		log.Fatalln("Failed to parse config file:", err)
	}

	// override defaults
	flowdockAPIKey = conf.FlowdockAPIKey
	if conf.StoragePath != "" {
		notificationStorage = conf.StoragePath
	}
	if conf.PingPrefix != 0 {
		pingPrefix = string(pingPrefix)
		slowPrefix = string(conf.PingPrefix)
		fastPrefix = slowPrefix + slowPrefix
		fasterPrefix = slowPrefix + fastPrefix
	}
	if conf.RolePrefix != 0 {
		rolePrefix = string(conf.RolePrefix)
	}

	// check that API key is given
	if flowdockAPIKey == "" {
		log.Fatal("An API key for Flowdock must be specified")
	}

	notifications := NewNotifications()
	restored, err := notifications.Restore(notificationStorage)
	log.Printf("Restored %d notifations from file '%s'", restored, notificationStorage)

	events := make(chan flowdock.Event)
	c := flowdock.NewClient(flowdockAPIKey)
	err = c.Connect(nil, events)
	if err != nil {
		panic(err)
	}

	users = NewUsers()
	for userID, _ := range c.Users {
		users.Add(c.Users[userID].Nick, userID)
	}
	users.Print()
	roles = NewRoles()
	restored, err = roles.Restore(roleStorage)
	log.Printf("Restored %d roles from file '%s'", restored, roleStorage)
	roles.Print()

	location, err := time.LoadLocation("Europe/Helsinki")
	if err != nil {
		log.Panic("Could not load timeszone info")
	}

	// build regex for matching pings
	notifRegex := regexp.MustCompile(fmt.Sprintf(`(\%s+)([\wåäö]+)`, pingPrefix))
	// and for matching role actions
	// syntax &[<rolename>][+|-|=][all|<username>](,<username>)
	roleRegex := regexp.MustCompile(fmt.Sprintf(`(\%s)([\wåäö]+)([+-=])([\wåäö,]+)`, rolePrefix))

	pingHelpMessage := "Notifybot does slow notifications."
	pingHelpMessage += " Create a slow notification for a person by doing " + slowPrefix + "<nick> or " + fastPrefix + "<nick> or " + fasterPrefix + "<nick>."
	pingHelpMessage += " The first will @<nick> the person the following day at 09:00 Finnish time."
	pingHelpMessage += " The others will notify <nick> after " + string(fastDelay) + " and " + string(fasterDelay) + " respectively."
	pingHelpMessage += " If the target is active in the thread, both all of notifications will be cleared."
	roleHelpMessage := "Notifybot supports roles."
	roleHelpMessage += " Add a person to a roly by doing " + rolePrefix + "<role>+<nick>. The role is created on the fly if it does not exists."
	roleHelpMessage += " Remove a person from a roly by doing " + rolePrefix + "<role>-<nick>."
	roleHelpMessage += " Several persons can be added or removed by doing e.g " + rolePrefix + "<role>+<nick1>,<nick2>."
	roleHelpMessage += " Replace persons in a role by doing " + rolePrefix + "<role>=<nick1>,<nick2>."
	roleHelpMessage += " List existing roles by doing " + rolePrefix + "list."
	roleHelpMessage += " List existing users in a role by doing " + rolePrefix + "<role>=list."

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
							message := fmt.Sprintf("@%v, slow ping from %v from [here](https://www.flowdock.com/app/walkbase/%s/messages/%d)", pingUser, strings.Title(notif.Pinger), flows[notif.Flow].APIName, notif.MessageID)
							var body []byte
							var err error
							if notif.Thread != "" {
								body, err = flowdock.SendMessageToFlowWithApiKey(flowdockAPIKey, notif.Flow, notif.Thread, message)
							}
							if err != nil {
								log.Panic(err)
							}
							log.Printf("%v\n", string(body))
							notifications.Delete(userID, threadID)
							notifications.Save(notificationStorage)
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
					notifications.Delete(event.UserID, event.ThreadID)
					nickClear := fmt.Sprintf("cleared-%s", c.Users[event.UserID].Nick)
					flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{nickClear})
				}

				if strings.HasPrefix(event.Content, pingPrefix+"help") {
					flowdock.SendMessageToFlowWithApiKey(flowdockAPIKey, event.Flow, event.ThreadID, pingHelpMessage)
				}
				if strings.HasPrefix(event.Content, rolePrefix+"help") {
					flowdock.SendMessageToFlowWithApiKey(flowdockAPIKey, event.Flow, event.ThreadID, roleHelpMessage)
				}
				if strings.HasPrefix(event.Content, rolePrefix+"list") {
					flowdock.SendCommentToFlowWithApiKey(flowdockAPIKey, event.Flow, event.ThreadID, roles.Roles())
				}

				for _, field := range notifRegex.FindAllStringSubmatch(event.Content, -1) {
					if len(field) < 2 {
						continue
					}
					possiblePrefix := field[1]
					possibleTarget := strings.ToLower(field[2])
					targets := []string{}
					if users.Exists(possibleTarget) {
						targets = append(targets, possibleTarget)
					} else if roles.Exists(possibleTarget) {
						targets = append(targets, roles[possibleTarget]...)
					}
					if len(targets) == 0 {
						log.Printf("User or Role '%s' does not exists", possibleTarget)
						continue
					}
					pinger := c.Users[event.UserID].Nick

					for _, target := range targets {
						notifyTime, notifyTag := createNotifyTimeAndTag(possiblePrefix, target, location)
						if !notifyTime.IsZero() {
							log.Printf("%s requested notification for %s at %v", pinger, target, notifyTime)
							notification := NewNotification(notifyTime, pinger, event.ThreadID, event.Flow, event.ID)
							notifications.Add(notification, users[target], event.ThreadID)
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{notifyTag})
							notifications.Save(notificationStorage)
						} else {
							log.Println("No time was set for notification")
						}
					}
				}
				for _, field := range roleRegex.FindAllStringSubmatch(event.Content, -1) {
					if len(field) < 4 {
						continue
					}
					possibleRoleName := field[2]
					possibleRoleAction := field[3]
					possibleRoleUsers := strings.Split(field[4], ",")
					roleUsers := []string{}
					// list users in role if requested
					if roles.Exists(possibleRoleName) && len(possibleRoleUsers) == 1 && possibleRoleUsers[0] == "list" && possibleRoleAction == "=" {
						flowdock.SendCommentToFlowWithApiKey(flowdockAPIKey, event.Flow, event.ThreadID, roles.Users(possibleRoleName))
						log.Println("listing users in role", possibleRoleName)
						continue
					}
					// filter unknown users
					for _, possibleRoleUser := range possibleRoleUsers {
						if users.Exists(possibleRoleUser) {
							roleUsers = append(roleUsers, possibleRoleUser)
						}
					}
					// if skip if no user was found
					if len(roleUsers) == 0 {
						log.Println("no users to add/remove/set")
						continue
					}

					switch possibleRoleAction {
					case "+":
						roleNotifyTag := roles.AddNotifyTag(possibleRoleName, roleUsers)
						roles.Add(possibleRoleName, roleUsers)
						roles.Save(roleStorage)
						if roleNotifyTag != "" {
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{roleNotifyTag})
						}
						break
					case "-":
						for _, roleUser := range roleUsers {
							if roleUser == "delete" {
								roleNotifyTag := roles.DeleteNotifyTag(possibleRoleName)
								roles.Delete(possibleRoleName)
								roles.Save(roleStorage)
								if roleNotifyTag != "" {
									flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{roleNotifyTag})
								}
								continue
							}
						}
						roleNotifyTag := roles.RemoveNotifyTag(possibleRoleName, roleUsers)
						roles.Remove(possibleRoleName, roleUsers)
						roles.Save(roleStorage)
						if roleNotifyTag != "" {
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{roleNotifyTag})
						}
						break
					case "=":
						roleNotifyTag := roles.SetNotifyTag(possibleRoleName, roleUsers)
						roles.Set(possibleRoleName, roleUsers)
						roles.Save(roleStorage)
						if roleNotifyTag != "" {
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{roleNotifyTag})
						}
						break
					default:
						log.Printf("Unknown role action '%s'", possibleRoleAction)
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
					notifications.Delete(event.UserID, messageID)
					nickClear := fmt.Sprintf("cleared-%s", c.Users[event.UserID].Nick)
					flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{nickClear})
				}

				if strings.HasPrefix(event.Content.Text, pingPrefix+"help") {
					flowdock.SendCommentToFlowWithApiKey(flowdockAPIKey, event.Flow, messageID, pingHelpMessage)
				}
				if strings.HasPrefix(event.Content.Text, rolePrefix+"help") {
					flowdock.SendCommentToFlowWithApiKey(flowdockAPIKey, event.Flow, messageID, roleHelpMessage)
				}
				if strings.HasPrefix(event.Content.Text, rolePrefix+"list") {
					flowdock.SendCommentToFlowWithApiKey(flowdockAPIKey, event.Flow, messageID, roles.Roles())
				}

				for _, field := range notifRegex.FindAllStringSubmatch(event.Content.Text, -1) {
					if len(field) < 2 {
						continue
					}
					possiblePrefix := field[1]
					possibleTarget := strings.ToLower(field[2])
					targets := []string{}
					if users.Exists(possibleTarget) {
						targets = append(targets, possibleTarget)
					} else if roles.Exists(possibleTarget) {
						targets = append(targets, roles[possibleTarget]...)
					}
					if len(targets) == 0 {
						log.Printf("User or Role '%s' does not exists", possibleTarget)
						continue
					}
					pinger := c.Users[event.UserID].Nick

					for _, target := range targets {
						notifyTime, notifyTag := createNotifyTimeAndTag(possiblePrefix, target, location)
						if !notifyTime.IsZero() {
							log.Printf("%s requested notification for %s at %v", pinger, target, notifyTime)
							notification := NewNotification(notifyTime, pinger, messageID, event.Flow, event.ID)
							notifications.Add(notification, users[target], event.Flow)
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{notifyTag})
							notifications.Save(notificationStorage)
						} else {
							log.Println("No time was set for notification")
						}
					}
				}
				for _, field := range roleRegex.FindAllStringSubmatch(event.Content.Text, -1) {
					if len(field) < 4 {
						continue
					}
					possibleRoleName := field[2]
					possibleRoleAction := field[3]
					possibleRoleUsers := strings.Split(field[4], ",")
					roleUsers := []string{}
					// list users in role if requested
					if roles.Exists(possibleRoleName) && len(possibleRoleUsers) == 1 && possibleRoleUsers[0] == "list" && possibleRoleAction == "=" {
						flowdock.SendCommentToFlowWithApiKey(flowdockAPIKey, event.Flow, messageID, roles.Users(possibleRoleName))
						log.Println("listing users in role", possibleRoleName)
						continue
					}
					// filter unknown users
					for _, possibleRoleUser := range possibleRoleUsers {
						if users.Exists(possibleRoleUser) {
							roleUsers = append(roleUsers, possibleRoleUser)
						}
					}
					// if skip if no user was found
					if len(roleUsers) == 0 {
						log.Println("no users to add/remove/set")
						continue
					}

					switch possibleRoleAction {
					case "+":
						roleNotifyTag := roles.AddNotifyTag(possibleRoleName, roleUsers)
						roles.Add(possibleRoleName, roleUsers)
						roles.Save(roleStorage)
						if roleNotifyTag != "" {
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{roleNotifyTag})
						}
						break
					case "-":
						for _, roleUser := range roleUsers {
							if roleUser == "delete" {
								roleNotifyTag := roles.DeleteNotifyTag(possibleRoleName)
								roles.Delete(possibleRoleName)
								roles.Save(roleStorage)
								if roleNotifyTag != "" {
									flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{roleNotifyTag})
								}
								continue
							}
						}
						roleNotifyTag := roles.RemoveNotifyTag(possibleRoleName, roleUsers)
						roles.Remove(possibleRoleName, roleUsers)
						roles.Save(roleStorage)
						if roleNotifyTag != "" {
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{roleNotifyTag})
						}
						break
					case "=":
						roleNotifyTag := roles.SetNotifyTag(possibleRoleName, roleUsers)
						roles.Set(possibleRoleName, roleUsers)
						roles.Save(roleStorage)
						if roleNotifyTag != "" {
							flowdock.EditMessageInFlowWithApiKey(flowdockAPIKey, org, flow, strconv.FormatInt(event.ID, 10), "", []string{roleNotifyTag})
						}
						break
					default:
						log.Printf("Unknown role action '%s'", possibleRoleAction)
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
						users.Add(c.Users[userID].Nick, userID)
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
