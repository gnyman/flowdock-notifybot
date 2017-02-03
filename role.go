package main

import (
	"fmt"
	"strings"
)

// Roles is a map of roles and users assinged to a role
type Roles map[string][]string

// NewRoles returns an empty map of roles
func NewRoles() Roles {
	return make(map[string][]string)
}

// Exists returns true the role exits
func (r Roles) Exists(role string) bool {
	if _, ok := r[strings.ToLower(role)]; ok {
		return true
	}
	return false
}

// Add adds users to a role
func (r Roles) Add(role string, users []string) {
	if r.Exists(role) {
		r[role] = append(r[role], users...)
		return
	}
	r[role] = users
}

// Remove removes users from a role
func (r Roles) Remove(role string, users []string) {
	if !r.Exists(role) {
		return
	}
	existing := r[role]
	unfiltered := []string{}
	// TODO: check if there's a go idiomatic way to remove
	// a subset from a slice
	for _, nick := range existing {
		keep := true
		for _, user := range users {
			if user == nick {
				keep = false
				break
			}
		}
		if keep {
			unfiltered = append(unfiltered, nick)
		}
	}
	r[role] = unfiltered
}

// Set sets users (replacing existing users) to a role
func (r Roles) Set(role string, users []string) {
	r[role] = users
}

// Print prints all roles and it's users
func (r Roles) Print() {
	for role, _ := range r {
		fmt.Printf("%s:%s\n", role, r.Users(role))
	}
}

// Users returns the users in role
func (r Roles) Users(role string) string {
	if !r.Exists(role) {
		return ""
	}

	return fmt.Sprintf("%v", r[role])
}
