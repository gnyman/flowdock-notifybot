// Package flowdock contains helpful structs and
// methods for dealing with Flowdock's RESTful API's.
// Structs are based on the message types defined here: https://www.flowdock.com/api/message-types
package flowdock

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// flowdockGET is a convenience function for performing
// GET requests against the Flowdock API
func flowdockGET(apiKey, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(apiKey, "BATMAN")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func pushMessage(flowAPIKey, message, sender string, threadID int64) error {
	v := url.Values{}
	v.Set("content", message)
	v.Set("external_user_name", sender)
	if threadID != 0 {
		v.Set("message_id", string(threadID))
	}

	pushURL := fmt.Sprintf("https://api.flowdock.com/v1/messages/chat/%s", flowAPIKey)
	resp, err := http.PostForm(pushURL, v)
	defer resp.Body.Close()

	if err != nil {
		return err
	}
	return nil
}

func SendMessageToFlowWithApiKey(apiKey, flowID, threadID, message string) ([]byte, error) {
	postURL := fmt.Sprintf("https://api.flowdock.com/messages")

	data := url.Values{}
	data.Set("flow", flowID)
	data.Set("content", message)
	data.Set("thread_id", threadID)
	data.Set("event", "message")

	req, err := http.NewRequest("POST", postURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(apiKey, "BATMAN")

	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func EditMessageInFlowWithApiKey(apiKey, organisation, flow, messageID, newMessage string, tags []string) ([]byte, error) {
	getMessageURL := fmt.Sprintf("https://api.flowdock.com/flows/%s/%s/messages/%s", organisation, flow, messageID)
	prevMessageBytes, err := flowdockGET(apiKey, getMessageURL)
	if err != nil {
		log.Printf("Error could not get message, error was %v", err)
	}
	prevMessage := MessageEvent{}
	err = json.Unmarshal(prevMessageBytes, &prevMessage)
	mergedTags := append(prevMessage.Tags, tags...)
	postURL := fmt.Sprintf("https://api.flowdock.com/flows/%s/%s/messages/%s", organisation, flow, messageID)
	log.Printf("Trying to edit %s", postURL)
	data := url.Values{}
	tags_string := strings.Join(mergedTags, ",")
	data.Set("tags", string(tags_string))

	req, err := http.NewRequest("PUT", postURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(apiKey, "BATMAN")

	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func SendCommentToFlowWithApiKey(apiKey, flowID, messageID, message string) ([]byte, error) {
	postURL := fmt.Sprintf("https://api.flowdock.com/comments")

	data := url.Values{}
	data.Set("flow", flowID)
	data.Set("content", message)
	data.Set("message", messageID)
	data.Set("event", "comment")

	req, err := http.NewRequest("POST", postURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(apiKey, "BATMAN")

	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// PushMessageToFlowWithKey can uses the Flowdock "Push" API to start
// a new thread in a flow using any pseudonym the client wishes. Useful
// for e.g implementing bots.
func PushMessageToFlowWithKey(flowAPIKey, message, sender string) error {
	return pushMessage(flowAPIKey, message, sender, 0)
}

// ReplyToThreadInFlowWithKey is similar to PushMessageToFlowWithKey
// except that it is used for replies rather than starting a new thread.
func ReplyToThreadInFlowWithKey(flowAPIKey, message, sender string, threadID int64) error {
	return pushMessage(flowAPIKey, message, sender, threadID)
}
