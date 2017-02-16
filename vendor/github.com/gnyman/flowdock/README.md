flowdock
========

A Go library for Flowdock's API


[![GoDoc](https://godoc.org/github.com/njern/flowdock?status.png)](https://godoc.org/github.com/njern/flowdock)
[![Build Status](https://travis-ci.org/njern/flowdock.png?branch=master)](https://travis-ci.org/njern/flowdock)<br>

## Usage


```go
    package main

    import (
        "github.com/njern/flowdock"
        "log"
    )

    const (
        flowdockAPIKey = "YOUR API KEY GOES HERE"
        someFlowAPIKey = "A FLOW SPECIFIC API KEY"
    )

    func main() {

        // Send a message to a specific flow using a pseudonym
	    flowdock.PushMessageToFlowWithKey(someFlowAPIKey, "It is I, Leclerc!", "Monsieur Roger LeClerc")


        events := make(chan flowdock.Event)
        c := flowdock.NewClient(flowdockAPIKey)
        err := c.Connect(nil, events)
        if err != nil {
            panic(err)
        }

        for {
            event := <-events

            switch event := event.(type) {
            case flowdock.MessageEvent:
                log.Printf("%s said (%s): '%s'", c.DetailsForUser(event.UserID).Nick, event.Flow, event.Content)
            case flowdock.CommentEvent:
                log.Printf("%s commented (%s): '%s'", c.DetailsForUser(event.UserID).Nick, event.Flow, event.Content.Text)
            case flowdock.MessageEditEvent:
                log.Printf("Looks like @%s just updated their previous message: '%s'. New message is '%s'", c.DetailsForUser(event.UserID).Nick, messageStore[event.Content.MessageID], event.Content.UpdatedMessage)
            case flowdock.UserActivityEvent:
                continue // Especially with > 10 people in your org, you will get MANY of these events.
            default:
                log.Printf("New event of type %T", event)
            }
        }
    }
```
