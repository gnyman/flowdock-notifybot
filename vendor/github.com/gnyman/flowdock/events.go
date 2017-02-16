package flowdock

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Event encompasses all the various Flowdock events
// that might arrive from the streaming API. The client
// is encouraged to use a type switch to handle the
// messages they are interested in and ignore the rest.
type Event interface{}

func unmarshalFlowdockJSONEvent(event []byte) (Event, error) {
	var m map[string]interface{}
	err := json.Unmarshal(event, &m)
	if err != nil {
		return nil, err
	}

	eventType := m["event"].(string)

	switch {
	case eventType == "message":
		e := MessageEvent{}
		err = json.Unmarshal(event, &e)
		if err != nil {
			log.Println("MessageEvent parsing error")
			return nil, err
		}
		return e, nil
	case eventType == "status":
		e := StatusEvent{}
		err = json.Unmarshal(event, &e)
		if err != nil {
			log.Println("StatusEvent parsing error")
			return nil, err
		}
		return e, nil
	case eventType == "message-edit":
		e := MessageEditEvent{}
		err = json.Unmarshal(event, &e)
		if err != nil {
			log.Println("MessageEditEvent parsing error")
			return nil, err
		}
		return e, nil
	case eventType == "comment":
		e := CommentEvent{}
		err = json.Unmarshal(event, &e)
		if err != nil {
			log.Println("CommentEvent parsing error")
			return nil, err
		}
		return e, nil
	case eventType == "tag-change":
		e := TagChangeEvent{}
		err = json.Unmarshal(event, &e)
		if err != nil {
			log.Println("TagChangeEvent parsing error")
			return nil, err
		}
		return e, nil
	case eventType == "activity.user":
		e := UserActivityEvent{}
		err = json.Unmarshal(event, &e)
		if err != nil {
			log.Println("UserActivityEvent parsing error")
			return nil, err
		}
		return e, nil
	case eventType == "file":
		e := FileEvent{}
		err = json.Unmarshal(event, &e)
		if err != nil {
			log.Println("FileEvent parsing error")
			return nil, err
		}
		return e, nil
	default:
		// Anything else is an ActionEvent... we hope
		e := ActionEvent{}
		err = json.Unmarshal(event, &e)
		if err != nil {
			log.Println("ActionEvent parsing error")
			return nil, err
		}
		return e, nil
	}
}

// A MessageEvent is sent when a user starts
// a new thread in a flow.
type MessageEvent struct {
	Tags        []string `json:"tags"`
	ID          int64    `json:"id"`
	Flow        string   `json:"flow"`
	Content     string   `json:"content"`
	Timestamp   int64    `json:"sent"`
	Attachments []string `json:"attachments"`
	UserID      string   `json:"user"`
	ThreadID    string   `json:"thread_id"`
}

// A UserIsTypingEvent is sent when a user
// starts typing in a flow. Several of these
// events may be sent while the user is typing
// the message.
type UserIsTypingEvent struct {
	Flow      string `json:"flow"`
	Timestamp int64  `json:"sent"`
	UserID    string `json:"user"`
}

// A StatusEvent is sent when a user changes their status.
type StatusEvent struct {
	Tags        []string `json:"tags"`
	ID          int64    `json:"id"`
	Flow        string   `json:"flow"`
	Content     string   `json:"content"`
	Timestamp   int64    `json:"sent"`
	Attachments []string `json:"attachments"`
	UserID      string   `json:"user"`
}

func (e *StatusEvent) String() string {
	return fmt.Sprintf("User with ID %s changed his/her status to %s at %v.", e.UserID, e.Content, time.Unix(e.Timestamp, 0))
}

// A CommentEvent is sent when a user comments
// on an item in the team inbox or on an existing
// message thread
type CommentEvent struct {
	Tags    []string `json:"tags"`
	ID      int64    `json:"id"`
	Flow    string   `json:"flow"`
	Content struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"content"`
	Timestamp   int64    `json:"sent"`
	Attachments []string `json:"attachments"`
	UserID      string   `json:"user"`
}

// An ActionEvent is sent by various activities, such as adding Twitter stream.
type ActionEvent struct {
	Type        string      `json:"event"`
	Tags        []string    `json:"tags"`
	ID          int64       `json:"id"`
	Flow        string      `json:"flow"`
	Content     interface{} `json:"content"` // Content can be either a string or an object
	Timestamp   int64       `json:"sent"`
	Attachments []string    `json:"attachments"`
	UserID      string      `json:"user"`
}

// A TagChangeEvent is sent when the tags of a message are changed.
type TagChangeEvent struct {
	Tags    []string `json:"tags"`
	ID      int64    `json:"id"`
	Flow    string   `json:"flow"`
	Content struct {
		Added     []string `json:"add"`
		Removed   []string `json:"remove"`
		MessageID int64    `json:"message"`
	} `json:"content"`
	Timestamp   int64    `json:"sent"`
	Attachments []string `json:"attachments"`
	UserID      string   `json:"user"`
}

// A MessageEditEvent is sent when the the content of a message is changed.
// Only messages of types 'message' and 'comment' can be edited.
type MessageEditEvent struct {
	Tags    []string `json:"tags"`
	ID      int64    `json:"id"`
	Flow    string   `json:"flow"`
	Content struct {
		UpdatedMessage string `json:"updated_content"`
		MessageID      int64  `json:"message"`
	} `json:"content"`
	Timestamp   int64    `json:"sent"`
	Attachments []string `json:"attachments"`
	UserID      string   `json:"user"`
}

// A UserActivityEvent is sent periodically by each user to let others know that they are online.
type UserActivityEvent struct {
	Tags    []string `json:"tags"`
	ID      int64    `json:"id"`
	Flow    string   `json:"flow"`
	Content struct {
		LastActivityTimestamp int64 `json:"last_activity"`
	} `json:"content"`
	Timestamp   int64    `json:"sent"`
	Attachments []string `json:"attachments"`
	UserID      string   `json:"user"`
}

// A FileEvent is sent when a file has been uploaded to the chat.
// content is an object that contains metadata about the uploaded file.
// The attachments field will contain a single attachment with the same data.
// In the metadata, the path field contains the REST API path of the file.
type FileEvent struct {
	Tags        []string                 `json:"tags"`
	ID          int64                    `json:"id"`
	Flow        string                   `json:"flow"`
	Content     map[string]interface{}   `json:"content"`
	Timestamp   int64                    `json:"sent"`
	Attachments []map[string]interface{} `json:"attachments"`
	UserID      string                   `json:"user"`
}

// FilePath returns the path from which the
// file specified can be fetched.
func (e *FileEvent) FilePath() string {
	path, ok := e.Content["path"]
	if ok {
		return path.(string)
	}

	return ""
}
