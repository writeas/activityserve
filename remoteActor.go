package activityserve

import (
	"fmt"
	"io/ioutil"

	"github.com/gologme/log"

	// "github.com/go-fed/activity/pub"
	// "github.com/go-fed/httpsig"

	"net/http"
	// "net/url"

	"encoding/json"

	"bytes"
)

// RemoteActor is a type that holds an actor
// that we want to interact with
type RemoteActor struct {
	iri, outbox, inbox, sharedInbox string
	info                            map[string]interface{}
}

// NewRemoteActor returns a remoteActor which holds
// all the info required for an actor we want to
// interact with (not essentially sitting in our instance)
func NewRemoteActor(iri string) (RemoteActor, error) {

	info, err := get(iri)
	if err != nil {
		log.Info("Couldn't get remote actor information")
		log.Info(err)
		return RemoteActor{}, err
	}

	outbox := info["outbox"].(string)
	inbox := info["inbox"].(string)
	var endpoints map[string]interface{}
	var sharedInbox string
	if info["endpoints"] != nil {
		endpoints = info["endpoints"].(map[string]interface{})
		sharedInbox = endpoints["sharedInbox"].(string)
	}

	return RemoteActor{
		iri:         iri,
		outbox:      outbox,
		inbox:       inbox,
		sharedInbox: sharedInbox,
	}, err
}

func (ra RemoteActor) getLatestPosts(number int) (map[string]interface{}, error) {
	return get(ra.outbox)
}

func get(iri string) (info map[string]interface{}, err error) {

	buf := new(bytes.Buffer)

	req, err := http.NewRequest("GET", iri, buf)
	if err != nil {
		log.Info(err)
		return
	}
	req.Header.Add("Accept", "application/activity+json; profile=\"https://www.w3.org/ns/activitystreams\"")
	req.Header.Add("User-Agent", userAgent+" "+version)
	req.Header.Add("Accept-Charset", "utf-8")

	resp, err := client.Do(req)

	if err != nil {
		log.Info("Cannot perform the request")
		log.Info(err)
		return
	}

	responseData, _ := ioutil.ReadAll(resp.Body)

	if !isSuccess(resp.StatusCode) {
		err = fmt.Errorf("GET request to %s failed (%d): %s\nResponse: %s \nHeaders: %s", iri, resp.StatusCode, resp.Status, FormatJSON(responseData), FormatHeaders(req.Header))
		log.Info(err)
		return
	}

	var e interface{}
	err = json.Unmarshal(responseData, &e)

	if err != nil {
		log.Info("something went wrong when unmarshalling the json")
		log.Info(err)
		return
	}
	info = e.(map[string]interface{})

	return
}

// GetInbox returns the inbox url of the actor
func (ra RemoteActor) GetInbox() string {
	return ra.inbox
}

// GetSharedInbox returns the inbox url of the actor
func (ra RemoteActor) GetSharedInbox() string {
	if ra.sharedInbox == "" {
		return ra.inbox
	}
	return ra.sharedInbox
}
