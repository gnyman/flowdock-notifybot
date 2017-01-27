package flowdock

import (
	"bytes"
	"log"
	"sync"

	"github.com/njern/httpstream"
)

const (
	flowdockAPIURL = "https://api.flowdock.com"
)

// A Client is a Flowdock API client. It should be created
// using NewClient() and provided with a valid API key.
type Client struct {
	apiKey         string
	streamClient   *httpstream.Client
	organizations  []Organization
	AvailableFlows []Flow // TODO: Change to map[ID]Flow
	Users          map[string]User
}

// NewClient creates a new Client and automatically fetches
// information about joined flows, known users, etc.
func NewClient(apiKey string) *Client {
	client := &Client{apiKey: apiKey}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		var err error
		client.Users, err = getUsers(apiKey)
		if err != nil {
			log.Printf("Failed to get users: %v", err)
		}
		wg.Done()
	}()

	go func() {
		var err error
		client.AvailableFlows, err = getFlows(apiKey)
		if err != nil {
			log.Printf("Failed to get flows: %v", err)
		}
		wg.Done()
	}()

	go func() {
		var err error
		client.organizations, err = getOrganizations(apiKey)
		if err != nil {
			log.Printf("Failed to get organizations: %v", err)
		}
		wg.Done()
	}()

	wg.Wait()

	return client
}

// Connect connects the client to the streaming API. Flowdock events will
// be sent back to the caller over the events channel, as will errors.
// If flows is nil, the stream will contain events from all the
// flows the user has joined.
// TODO: pass online state (active / idle / offline)
// TODO: Flag for whether to receive priv messages or not (now defaulting to not)
func (c *Client) Connect(flows []Flow, events chan Event) error {
	stream := make(chan []byte, 1024) // Incoming (raw) data channel
	done := make(chan error)          // Error channel
	flowURL := flowStreamURL(c, flows)

	// Set up the stream. Note we need to set a random password string ("BATMAN") or things will break.
	c.streamClient = httpstream.NewBasicAuthClient(c.apiKey, "BATMAN", func(line []byte) {
		stream <- line
	})

	// Initialize the connection
	err := c.streamClient.Connect(flowURL, done)
	if err != nil {
		return err
	}

	// Fire up a goroutine that will listen to the stream
	// and pass events back to the client.
	go func() {
		for {
			select {
			case event := <-stream:
				if bytes.Equal(event, []byte{10}) {
					log.Printf("Keepalive, ignore...")
					continue
				}
				parsedEvent, err := unmarshalFlowdockJSONEvent(event)
				if err != nil {
					log.Printf("Error could not unmarshal json, error %v, raw json: %v", err, string(event))
					log.Printf("RAW: %v", event)
					log.Printf("String: %v", string(event))
					events <- err
				} else {
					events <- parsedEvent
				}

			case err := <-done:
				if err != nil {
					log.Printf("Stream error, error %v", err)
					log.Printf("Reconnecting?")
					c.streamClient.Connect(flowURL, done)
					// TODO: Actually handle errors instead of just closing the channel
					events <- err
					//close(events)
				}
			}
		}
	}()

	return nil
}

// DetailsForUser returns a User object for the given user ID.
func (c *Client) DetailsForUser(id string) User {
	return c.Users[id]
}

// DetailsForFlow returns a Flow object for the given Flow ID
// or nil if the client can't access details for that flow.
func (c *Client) DetailsForFlow(id string) *Flow {
	for _, flow := range c.AvailableFlows {
		if flow.ID == id {
			return &flow
		}
	}

	return nil
}

// SendMessage starts a new thread in the specified Flow
// TODO: Implement this
func SendMessage(flow Flow, message string) error {
	return nil
}

// SendReply replies to an existing thread
// TODO: Implement this
func SendReply(flow Flow, reply string, threadID int64) error {
	return nil
}

// flowStreamURL creates the complete URL used to connect
// to the streaming API endpoint, including the flows filter.
func flowStreamURL(c *Client, flows []Flow) string {
	if flows == nil {
		// Add all the flows!
		flows = c.AvailableFlows
	}

	flowURL := "https://stream.flowdock.com/flows?filter="
	for i, flow := range flows {
		if i == len(flows)-1 {
			// Special case; Last item - no comma.
			flowURL = flowURL + flow.Organization.APIName + "/" + flow.APIName
		} else {
			flowURL = flowURL + flow.Organization.APIName + "/" + flow.APIName + ","
		}
	}
	return flowURL
}
