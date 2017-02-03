package main

import (
	"testing"
)

func TestAddRemoveSet(t *testing.T) {

	roles := NewRoles()

	if len(roles) != 0 {
		t.Errorf("roles should be empty")
	}

	// add
	roles.Add("foo", []string{"nick", "nack"})
	roles.Add("bar", []string{"nack"})
	roles.Add("foo", []string{"foo"})
	roles.Add("foo", []string{"bar"})
	if len(roles) != 2 || len(roles["foo"]) != 4 || len(roles["bar"]) != 1 {
		t.Errorf("Add failed")
	}

	// remove
	roles.Remove("foo", []string{"foo", "bar"})
	roles.Remove("foo", []string{"foobar", "foobar"}) // users not in role
	roles.Remove("foobar", []string{"foo", "bar"})    // role does not exist
	if len(roles) != 2 || len(roles["foo"]) != 2 || len(roles["bar"]) != 1 {
		t.Errorf("Remove failed")
	}

	// set
	roles.Set("bar", []string{"foo", "bar"})
	if len(roles) != 2 || len(roles["foo"]) != 2 || len(roles["bar"]) != 2 {
		t.Errorf("Set failed")
	}
}
