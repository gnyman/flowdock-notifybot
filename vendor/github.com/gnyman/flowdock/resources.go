package flowdock

import (
	"encoding/json"
	"strconv"
)

// User resource as seen by GET /users
type User struct {
	ID                    int64  `json:"id"`
	Name                  string `json:"name"`
	Nick                  string `json:"nick"`
	AvatarURL             string `json:"avatar"`
	Status                string `json:"status"`
	LastActivityTimestamp int64  `json:"last_activity"`
	LastPingTimestamp     int64  `json:"last_ping"`
	Email                 string `json:"email"`
}

// getUsers fetches users from the GET /users endpoint
func getUsers(apiKey string) (map[string]User, error) {
	body, err := flowdockGET(apiKey, flowdockAPIURL+"/users")
	if err != nil {
		return nil, err
	}

	var users []User
	err = json.Unmarshal(body, &users)
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]User)
	for _, user := range users {
		userID := strconv.FormatInt(user.ID, 10)
		userMap[userID] = user
	}

	return userMap, nil
}

// Organization resource, as seen by GET /organization
type Organization struct {
	ID      int64  `json:"id"`
	APIName string `json:"parameterized_name"`
	Name    string `json:"name"`
	APIURL  string `json:"url"`
	Users   []User `json:"users"` // Maps user ID's to user objects.
}

// getOrganizations fetches organisations from the GET /organizations endpoint
func getOrganizations(apiKey string) ([]Organization, error) {
	body, err := flowdockGET(apiKey, flowdockAPIURL+"/organizations")
	if err != nil {
		return nil, err
	}

	var orgs []Organization
	err = json.Unmarshal(body, &orgs)
	if err != nil {
		return nil, err
	}

	return orgs, nil
}

// Flow resource, as seen by GET /flows
type Flow struct {
	ID           string       `json:"id"`
	APIURL       string       `json:"url"`
	WebURL       string       `json:"web_url"`
	Name         string       `json:"name"`
	APIName      string       `json:"parameterized_name"`
	Organization Organization `json:"organization"`
}

// getFlows fetches Flows from the GET /flows endpoint
func getFlows(apiKey string) ([]Flow, error) {
	body, err := flowdockGET(apiKey, flowdockAPIURL+"/flows")
	if err != nil {
		return nil, err
	}

	var flows []Flow
	err = json.Unmarshal(body, &flows)
	if err != nil {
		return nil, err
	}

	return flows, nil
}
