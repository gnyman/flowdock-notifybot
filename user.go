package main

import (
	"fmt"
	"strings"
)

// Users is a map of user nickames and IDs
type Users map[string]string

// NewUsers returns an empty map of users
func NewUsers() Users {
	return make(map[string]string)
}

// Exists returns true the nick exists in Users
func (u Users) Exists(nick string) bool {
	if _, ok := u[strings.ToLower(nick)]; ok {
		return true
	}
	return false
}

// Add adds user with nick (lower cased) to users
func (u Users) Add(nick, id string) {
	u[strings.ToLower(nick)] = id
}

// Print prints all users
func (u Users) Print() {
	for nick, id := range u {
		fmt.Println("%s:%s", nick, id)
	}
}
