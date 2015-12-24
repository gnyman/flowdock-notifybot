package main

import (
	"fmt"
	"testing"
)

func TestNextWorkdayNine(t *testing.T) {
	fmt.Printf("%v", NextWorkdayAtNine())
	t.Fail()
}
