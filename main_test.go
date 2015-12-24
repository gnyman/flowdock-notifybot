package main

import (
	"testing"
)

func TestNextWorkdayNine(t *testing.T) {
    nextWorkDayAtNine := NextWorkdayAtNine()
    hour,min,sec := nextWorkDayAtNine.Clock()
    if hour != 9 && min != 0 && sec != 0 {
        t.Error("Expected NextWorkdayAtNine to be at 9:00:00")
    }
}
