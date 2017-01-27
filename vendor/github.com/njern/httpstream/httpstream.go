package httpstream

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

// Client for connecting to a stream
type Client struct {
	username        string         // Basic auth username
	password        string         // Basic auth password
	shouldClose     bool           // Set to close stream cleanly
	shouldCloseLock sync.Mutex     // Avoid data races
	activeConn      *http.Response // Active connection reference
	handler         func([]byte)   // The function called with the content of received lines
}

// NewClient returns a new HTTP streaming client with default values
func NewClient(handler func([]byte)) *Client {
	return &Client{
		handler: handler,
	}
}

// NewBasicAuthClient returns a new HTTP streaming client with HTTP Basic authentication.
func NewBasicAuthClient(username, password string, handler func([]byte)) *Client {
	return &Client{
		username: username,
		password: password,
		handler:  handler,
	}
}

// Connect connects the client to a streaming HTTP endpoint.
func (c *Client) Connect(url string, done chan error) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// If HTTP basic auth is enabled..
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	} else if resp == nil {
		return errors.New("no response")
	} else if resp.StatusCode != 200 {
		err = fmt.Errorf("HTTP Error: %d for url: %s", resp.Status, url)
	}

	// If we have an old connection, close it.
	if c.activeConn != nil {
		c.activeConn.Body.Close()
	}

	// Set the new connection as the client's active connection
	c.activeConn = resp
	go c.readStream(done)
	return nil
}

// Close the stream cleanly. The client will return
// nil on the done channel once the stream has been closed.
func (c *Client) Close() {
	c.shouldCloseLock.Lock()
	c.shouldClose = true
	c.shouldCloseLock.Unlock()
}

// Reads the stream continously. Returns error(s) on the done chan for the caller to deal with.
func (c *Client) readStream(done chan error) {
	reader := bufio.NewReader(c.activeConn.Body)

	for {
		if c.shouldBeClosed() {
			c.activeConn.Body.Close()
			c.activeConn = nil
			done <- nil
			return
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			c.activeConn.Body.Close()
			c.activeConn = nil
			done <- err
			return
		}

		if len(line) == 0 {
			continue
		}

		c.handler(line)
	}
}

// Safely check if stream should close
func (c *Client) shouldBeClosed() bool {
	c.shouldCloseLock.Lock()
	r := c.shouldClose
	c.shouldCloseLock.Unlock()

	return r
}
