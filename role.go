package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// Roles is a map of roles and users assinged to a role
type Roles map[string][]string

// NewRoles returns an empty map of roles
func NewRoles() Roles {
	return make(map[string][]string)
}

// Restore restores saved roles from file
func (r Roles) Restore(file string) (int, error) {
	if _, err := os.Stat(file); err == nil {
		rawData, err := ioutil.ReadFile(file)
		if err != nil {
			return 0, fmt.Errorf("Error could not restore roles because could not read file :-(")
		}
		buffer := bytes.NewBuffer(rawData)
		dec := gob.NewDecoder(buffer)

		err = dec.Decode(&r)
		if err != nil {
			return 0, fmt.Errorf("Error could not decode %v", dec)
		}
		return len(r), nil
	}
	return 0, nil
}

// Save saves roles to file
func (r Roles) Save(file string) error {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(r)
	if err != nil {
		return fmt.Errorf("Error could not save the roles")
	}

	ioutil.WriteFile(file, buffer.Bytes(), 0600)
	return nil
}

// Exists returns true the role exits
func (r Roles) Exists(role string) bool {
	if _, ok := r[strings.ToLower(role)]; ok {
		return true
	}
	return false
}

// userExistsInRole returns true if the user already belongs
// to role
func (r Roles) userExistsInRole(role, user string) bool {
	if r.Exists(role) {
		for _, u := range r[role] {
			if u == user {
				return true
			}
		}
	}
	return false
}

// Delete deletes role if it exists
func (r Roles) Delete(role string) {
	delete(r, role)
}

// Add adds users to a role
func (r Roles) Add(role string, users []string) {
	if !r.Exists(role) {
		r.Set(role, users)
		return
	}

	for _, user := range users {
		if !r.userExistsInRole(role, user) {
			r[role] = append(r[role], user)
		}
	}
}

// Remove removes users from a role
func (r Roles) Remove(role string, users []string) {
	if !r.Exists(role) {
		return
	}
	u := []string{}
	// TODO: check if there's a go idiomatic way to remove
	// a subset from a slice
	for _, user := range users {
		if r.userExistsInRole(role, user) {
			continue
		}
		u = append(u, user)
	}
	r[role] = u
}

// Set sets users (replacing existing users) to a role
func (r Roles) Set(role string, users []string) {
	r[role] = []string{}

	for _, user := range users {
		if !r.userExistsInRole(role, user) {
			r[role] = append(r[role], user)
		}
	}
}

// Print prints all roles and it's users
func (r Roles) Print() {
	for role, _ := range r {
		fmt.Printf("%s:%s\n", role, r.Users(role))
	}
}

// Roles return the existing roles
func (r Roles) Roles() string {
	roles := []string{}
	for role := range r {
		roles = append(roles, role)
	}
	return fmt.Sprintf("%v", roles)
}

// Users returns the users in role
func (r Roles) Users(role string) string {
	if !r.Exists(role) {
		return ""
	}

	return fmt.Sprintf("%v", r[role])
}

// AddNotifyTag returns a message of what will be added
func (r Roles) AddNotifyTag(role string, users []string) string {
	if !r.Exists(role) {
		return fmt.Sprintf("role %s added with users: %s", role, users)
	}
	newUsers := []string{}
	for _, user := range users {
		if r.userExistsInRole(role, user) {
			continue
		}
		newUsers = append(newUsers, user)
	}

	if len(newUsers) == 0 {
		return ""
	}
	return fmt.Sprintf("users %s added to role %s", newUsers, role)
}

// RemoveNotifyTag returns a message of what will be removed
func (r Roles) RemoveNotifyTag(role string, users []string) string {
	if !r.Exists(role) {
		return ""
	}

	oldUsers := []string{}
	for _, user := range users {
		if r.userExistsInRole(role, user) {
			oldUsers = append(oldUsers, user)
		}
	}

	if len(oldUsers) == 0 {
		return ""
	}
	return fmt.Sprintf("users %s removed from role %s", oldUsers, role)
}

// SetNotifyTag returns a message of what will be changed
func (r Roles) SetNotifyTag(role string, users []string) string {
	if !r.Exists(role) {
		return fmt.Sprintf("role %s added with users: %s", role, users)
	}

	return fmt.Sprintf("users %s set to role %s", users, role)
}

// DeleteNotifyTag returns a message of what will be deleted
func (r Roles) DeleteNotifyTag(role string) string {
	if !r.Exists(role) {
		return ""
	}

	return fmt.Sprintf("role %s deleted", role)
}
