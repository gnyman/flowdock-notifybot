package main

import (
	"reflect"
	"testing"
)

func TestAddRemoveSetDelete(t *testing.T) {

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

	// delete
	roles.Delete("bar")
	if len(roles) != 1 || len(roles["foo"]) != 2 {
		t.Errorf("Delete failed")
	}
}

func TestRolesStoreAndRestore(t *testing.T) {
	roles := NewRoles()

	roles.Add("role1", []string{"user1", "user2"})
	roles.Add("role2", []string{"user1"})
	roles.Add("role3", []string{"user2", "user3"})

	file := "/tmp/test-flowdock-roles.gob"
	err := roles.Save(file)
	if err != nil {
		t.Fatal(err)
	}

	restoredRoles := NewRoles()
	restored, err := restoredRoles.Restore(file)
	if err != nil {
		t.Fatal(err)
	}

	if restored != 3 {
		t.Errorf("Restore: wanted %d, got %d", 3, restored)
	}
	if !reflect.DeepEqual(roles, restoredRoles) {
		t.Errorf("wanted %v", roles)
		t.Errorf("got %v", restoredRoles)
	}
}

func TestTags(t *testing.T) {
	roles := NewRoles()

	var msg string

	// add
	msg = roles.AddNotifyTag("foobar", []string{"foo", "bar"})
	if msg != "role foobar added with users: [foo bar]" {
		t.Errorf("add tag failed, got: '%s'", msg)
	}
	roles.Add("foobar", []string{"foo", "bar"})
	msg = roles.AddNotifyTag("foobar", []string{"foobar"})
	if msg != "users [foobar] added to role foobar" {
		t.Errorf("add tag failed, got: '%s'", msg)
	}
	roles.Add("foobar", []string{"foobar"})
	msg = roles.AddNotifyTag("foobar", []string{"foobar"})
	if msg != "" {
		t.Errorf("add tag failed, got: '%s'", msg)
	}

	// remove
	msg = roles.RemoveNotifyTag("fozbaz", []string{"foobar"})
	if msg != "" {
		t.Errorf("remove tag failed, got: '%s'", msg)
	}
	msg = roles.RemoveNotifyTag("foobar", []string{"fozbaz"})
	if msg != "" {
		t.Errorf("remove tag failed, got: '%s'", msg)
	}
	msg = roles.RemoveNotifyTag("foobar", []string{"foo", "bar"})
	if msg != "users [foo bar] removed from role foobar" {
		t.Errorf("remove tag failed, got: '%s'", msg)
	}

	// set
	msg = roles.SetNotifyTag("foo", []string{"bar", "foo"})
	if msg != "role foo added with users: [bar foo]" {
		t.Errorf("set tag failed, got: '%s'", msg)
	}
	roles.Set("foo", []string{"bar", "foo"})
	msg = roles.SetNotifyTag("foobar", []string{"foo", "bar"})
	if msg != "users [foo bar] set to role foobar" {
		t.Errorf("set tag failed, got: '%s'", msg)
	}
	roles.Set("foobar", []string{"foo", "bar"})

	// delete
	msg = roles.DeleteNotifyTag("fozbaz")
	if msg != "" {
		t.Errorf("delete tag failed, got: '%s'", msg)
	}
	msg = roles.DeleteNotifyTag("foo")
	if msg != "role foo deleted" {
		t.Errorf("delete tag failed, got: '%s'", msg)
	}
}
