package activityserve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gologme/log"

	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"github.com/dchest/uniuri"
	"github.com/go-fed/httpsig"
)

// Actor represents a local actor we can act on
// behalf of.
type Actor struct {
	name, summary, actorType, iri  string
	followersIRI                   string
	nuIri                          *url.URL
	followers, following, rejected map[string]interface{}
	posts                          map[int]map[string]interface{}
	publicKey                      crypto.PublicKey
	privateKey                     crypto.PrivateKey
	publicKeyPem                   string
	privateKeyPem                  string
	publicKeyID                    string
	OnFollow                       func(map[string]interface{})
}

// ActorToSave is a stripped down actor representation
// with exported properties in order for json to be
// able to marshal it.
// see https://stackoverflow.com/questions/26327391/json-marshalstruct-returns
type ActorToSave struct {
	Name, Summary, ActorType, IRI, PublicKey, PrivateKey string
	Followers, Following, Rejected                       map[string]interface{}
}

// MakeActor returns a new local actor we can act
// on behalf of
func MakeActor(name, summary, actorType string) (Actor, error) {
	followers := make(map[string]interface{})
	following := make(map[string]interface{})
	rejected := make(map[string]interface{})
	followersIRI := baseURL + name + "/followers"
	publicKeyID := baseURL + name + "#main-key"
	iri := baseURL + name
	nuIri, err := url.Parse(iri)
	if err != nil {
		log.Info("Something went wrong when parsing the local actor uri into net/url")
		return Actor{}, err
	}
	actor := Actor{
		name:         name,
		summary:      summary,
		actorType:    actorType,
		iri:          iri,
		nuIri:        nuIri,
		followers:    followers,
		following:    following,
		rejected:     rejected,
		followersIRI: followersIRI,
		publicKeyID:  publicKeyID,
	}

	// set auto accept by default (this could be a configuration value)
	actor.OnFollow = func(activity map[string]interface{}) { actor.Accept(activity) }

	// create actor's keypair
	rng := rand.Reader
	privateKey, err := rsa.GenerateKey(rng, 2048)
	publicKey := privateKey.PublicKey

	actor.publicKey = publicKey
	actor.privateKey = privateKey

	// marshal the crypto to pem
	privateKeyDer := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyDer,
	}
	actor.privateKeyPem = string(pem.EncodeToMemory(&privateKeyBlock))

	publicKeyDer, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		log.Info("Can't marshal public key")
		return Actor{}, err
	}

	publicKeyBlock := pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   publicKeyDer,
	}
	actor.publicKeyPem = string(pem.EncodeToMemory(&publicKeyBlock))

	err = actor.save()
	if err != nil {
		return actor, err
	}

	return actor, nil
}

// GetOutboxIRI returns the outbox iri in net/url
func (a *Actor) GetOutboxIRI() *url.URL {
	iri := a.iri + "/outbox"
	nuiri, _ := url.Parse(iri)
	return nuiri
}

// LoadActor searches the filesystem and creates an Actor
// from the data in name.json
func LoadActor(name string) (Actor, error) {
	// make sure our users can't read our hard drive
	if strings.ContainsAny(name, "./ ") {
		log.Info("Illegal characters in actor name")
		return Actor{}, errors.New("Illegal characters in actor name")
	}
	jsonFile := storage + slash + "actors" + slash + name + slash + name + ".json"
	fileHandle, err := os.Open(jsonFile)
	if os.IsNotExist(err) {
		log.Info("We don't have this kind of actor stored")
		return Actor{}, err
	}
	byteValue, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		log.Info("Error reading actor file")
		return Actor{}, err
	}
	jsonData := make(map[string]interface{})
	json.Unmarshal(byteValue, &jsonData)

	nuIri, err := url.Parse(jsonData["IRI"].(string))
	if err != nil {
		log.Info("Something went wrong when parsing the local actor uri into net/url")
		return Actor{}, err
	}

	// publicKeyNewLines := strings.ReplaceAll(jsonData["PublicKey"].(string), "\\n", "\n")
	// privateKeyNewLines := strings.ReplaceAll(jsonData["PrivateKey"].(string), "\\n", "\n")

	publicKeyDecoded, rest := pem.Decode([]byte(jsonData["PublicKey"].(string)))
	if publicKeyDecoded == nil {
		log.Info(rest)
		panic("failed to parse PEM block containing the public key")
	}
	publicKey, err := x509.ParsePKIXPublicKey(publicKeyDecoded.Bytes)
	if err != nil {
		log.Info("Can't parse public keys")
		log.Info(err)
		return Actor{}, err
	}
	privateKeyDecoded, rest := pem.Decode([]byte(jsonData["PrivateKey"].(string)))
	if privateKeyDecoded == nil {
		log.Info(rest)
		panic("failed to parse PEM block containing the private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyDecoded.Bytes)
	if err != nil {
		log.Info("Can't parse private keys")
		log.Info(err)
		return Actor{}, err
	}

	actor := Actor{
		name:          name,
		summary:       jsonData["Summary"].(string),
		actorType:     jsonData["ActorType"].(string),
		iri:           jsonData["IRI"].(string),
		nuIri:         nuIri,
		followers:     jsonData["Followers"].(map[string]interface{}),
		following:     jsonData["Following"].(map[string]interface{}),
		rejected:      jsonData["Rejected"].(map[string]interface{}),
		publicKey:     publicKey,
		privateKey:    privateKey,
		publicKeyPem:  jsonData["PublicKey"].(string),
		privateKeyPem: jsonData["PrivateKey"].(string),
		followersIRI:  baseURL + name + "/followers",
		publicKeyID:   baseURL + name + "#main-key",
	}

	actor.OnFollow = func(activity map[string]interface{}) { actor.Accept(activity) }

	return actor, nil
}

// GetActor attempts to LoadActor and if it doesn't exist
// creates one
func GetActor(name, summary, actorType string) (Actor, error) {
	actor, err := LoadActor(name)

	if err != nil {
		log.Info("Actor doesn't exist, creating...")
		actor, err = MakeActor(name, summary, actorType)
		if err != nil {
			log.Info("Can't create actor!")
			return Actor{}, err
		}
	}

	// if the info provided for the actor is different
	// from what the actor has, edit the actor
	save := false
	if summary != actor.summary {
		actor.summary = summary
		save = true
	}
	if actorType != actor.actorType {
		actor.actorType = actorType
		save = true
	}
	// if anything changed write it to disk
	if save {
		actor.save()
	}

	return actor, nil
}

// func LoadActorFromIRI(iri string) a Actor{

// }

// save the actor to file
func (a *Actor) save() error {

	// check if we already have a directory to save actors
	// and if not, create it
	dir := storage + slash + "actors" + slash + a.name + slash + "items"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	actorToSave := ActorToSave{
		Name:       a.name,
		Summary:    a.summary,
		ActorType:  a.actorType,
		IRI:        a.iri,
		Followers:  a.followers,
		Following:  a.following,
		Rejected:   a.rejected,
		PublicKey:  a.publicKeyPem,
		PrivateKey: a.privateKeyPem,
	}

	actorJSON, err := json.MarshalIndent(actorToSave, "", "\t")
	if err != nil {
		log.Info("error Marshalling actor json")
		return err
	}
	// log.Info(actorToSave)
	// log.Info(string(actorJSON))
	err = ioutil.WriteFile(storage+slash+"actors"+slash+a.name+slash+a.name+".json", actorJSON, 0644)
	if err != nil {
		log.Printf("WriteFileJson ERROR: %+v", err)
		return err
	}

	return nil
}

func (a *Actor) whoAmI() string {
	return `{"@context":	"https://www.w3.org/ns/activitystreams",
	"type": "` + a.actorType + `",
	"id": "` + baseURL + a.name + `",
	"name": "` + a.name + `",
	"preferredUsername": "` + a.name + `",
	"summary": "` + a.summary + `",
	"inbox": "` + baseURL + a.name + `/inbox",
	"outbox": "` + baseURL + a.name + `/outbox",
	"followers": "` + baseURL + a.name + `/peers/followers",
	"following": "` + baseURL + a.name + `/peers/following",
	"publicKey": {
		"id": "` + baseURL + a.name + `#main-key",
		"owner": "` + baseURL + a.name + `",
		"publicKeyPem": "` + strings.ReplaceAll(a.publicKeyPem, "\n", "\\n") + `"
	  }
	}`
}

func (a *Actor) newItemID() (hash string, url string) {
	hash = uniuri.New()
	return hash, baseURL + a.name + "/item/" + hash
}

func (a *Actor) newID() (hash string, url string) {
	hash = uniuri.New()
	return hash, baseURL + a.name + "/" + hash
}

// TODO Reply(content string, inReplyTo string)

// ReplyNote sends a note to a specific actor in reply to
// a post
//TODO

// DM sends a direct message to a user
// TODO

// CreateNote posts an activityPub note to our followers
//
func (a *Actor) CreateNote(content, inReplyTo string) {
	// for now I will just write this to the outbox
	hash, id := a.newItemID()
	create := make(map[string]interface{})
	note := make(map[string]interface{})
	create["@context"] = context()
	create["actor"] = baseURL + a.name
	create["cc"] = a.followersIRI
	create["id"] = id
	create["object"] = note
	note["attributedTo"] = baseURL + a.name
	note["cc"] = a.followersIRI
	note["content"] = content
	if inReplyTo != "" {
		note["inReplyTo"] = inReplyTo
	}
	note["id"] = id
	note["published"] = time.Now().Format(time.RFC3339)
	note["url"] = create["id"]
	note["type"] = "Note"
	note["to"] = "https://www.w3.org/ns/activitystreams#Public"
	create["published"] = note["published"]
	create["type"] = "Create"
	go a.sendToFollowers(create)
	err := a.saveItem(hash, create)
	if err != nil {
		log.Info("Could not save note to disk")
	}
	err = a.appendToOutbox(id)
	if err != nil {
		log.Info("Could not append Note to outbox.txt")
	}
}

// saveItem saves an activity to disk under the actor and with the id as
// filename
func (a *Actor) saveItem(hash string, content map[string]interface{}) error {
	JSON, _ := json.MarshalIndent(content, "", "\t")

	dir := storage + slash + "actors" + slash + a.name + slash + "items"
	err := ioutil.WriteFile(dir+slash+hash+".json", JSON, 0644)
	if err != nil {
		log.Printf("WriteFileJson ERROR: %+v", err)
		return err
	}
	return nil
}

func (a *Actor) loadItem(hash string) (item map[string]interface{}, err error) {
	dir := storage + slash + "actors" + slash + a.name + slash + "items"
	jsonFile := dir + slash + hash + ".json"
	fileHandle, err := os.Open(jsonFile)
	if os.IsNotExist(err) {
		log.Info("We don't have this item stored")
		return
	}
	byteValue, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		log.Info("Error reading item file")
		return
	}
	json.Unmarshal(byteValue, &item)

	return
}

// send is here for backward compatibility and maybe extra pre-processing
// not always required
func (a *Actor) send(content map[string]interface{}, to *url.URL) (err error) {
	return a.signedHTTPPost(content, to.String())
}

// getPeers gets followers or following depending on `who`
func (a *Actor) getPeers(page int, who string) (response []byte, err error) {
	// if there's no page parameter mastodon displays an
	// OrderedCollection with info of where to find orderedCollectionPages
	// with the actual information. We are mirroring that behavior

	var collection map[string]interface{}
	if who == "followers" {
		collection = a.followers
	} else if who == "following" {
		collection = a.following
	} else {
		return nil, errors.New("cannot find collection" + who)
	}
	themap := make(map[string]interface{})
	themap["@context"] = "https://www.w3.org/ns/activitystreams"
	if page == 0 {
		themap["first"] = baseURL + a.name + "/" + who + "?page=1"
		themap["id"] = baseURL + a.name + "/" + who
		themap["totalItems"] = strconv.Itoa(len(collection))
		themap["type"] = "OrderedCollection"
	} else if page == 1 { // implement pagination
		themap["id"] = baseURL + a.name + who + "?page=" + strconv.Itoa(page)
		items := make([]string, 0, len(collection))
		for k := range collection {
			items = append(items, k)
		}
		themap["orderedItems"] = items
		themap["partOf"] = baseURL + a.name + "/" + who
		themap["totalItems"] = len(collection)
		themap["type"] = "OrderedCollectionPage"
	}
	response, _ = json.Marshal(themap)
	return
}

// GetFollowers returns a list of people that follow us
func (a *Actor) GetFollowers(page int) (response []byte, err error) {
	return a.getPeers(page, "followers")
}

// GetFollowing returns a list of people that we follow
func (a *Actor) GetFollowing(page int) (response []byte, err error) {
	return a.getPeers(page, "following")
}

func (a *Actor) signedHTTPPost(content map[string]interface{}, to string) (err error) {
	b, err := json.Marshal(content)
	if err != nil {
		log.Info("Can't marshal JSON")
		log.Info(err)
		return
	}
	postSigner, _, _ := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, []string{"(request-target)", "date", "host", "digest"}, httpsig.Signature)

	byteCopy := make([]byte, len(b))
	copy(byteCopy, b)
	buf := bytes.NewBuffer(byteCopy)
	req, err := http.NewRequest("POST", to, buf)
	if err != nil {
		log.Info(err)
		return
	}

	// I prefer to deal with strings and just parse to net/url if and when
	// needed, even if here we do one extra round trip
	iri, err := url.Parse(to)
	if err != nil {
		log.Error("cannot parse url for POST, check your syntax")
		return err
	}
	req.Header.Add("Accept-Charset", "utf-8")
	req.Header.Add("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	req.Header.Add("User-Agent", userAgent+" "+version)
	req.Header.Add("Host", iri.Host)
	req.Header.Add("Accept", "application/activity+json")
	sum := sha256.Sum256(b)
	req.Header.Add("Digest",
		fmt.Sprintf("SHA-256=%s",
			base64.StdEncoding.EncodeToString(sum[:])))
	err = postSigner.SignRequest(a.privateKey, a.publicKeyID, req)
	if err != nil {
		log.Info(err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Info(err)
		return
	}
	defer resp.Body.Close()
	if !isSuccess(resp.StatusCode) {
		responseData, _ := ioutil.ReadAll(resp.Body)
		err = fmt.Errorf("POST request to %s failed (%d): %s\nResponse: %s \nRequest: %s \nHeaders: %s", to, resp.StatusCode, resp.Status, FormatJSON(responseData), FormatJSON(byteCopy), FormatHeaders(req.Header))
		log.Info(err)
		return
	}
	responseData, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("POST request to %s succeeded (%d): %s \nResponse: %s \nRequest: %s \nHeaders: %s", to, resp.StatusCode, resp.Status, FormatJSON(responseData), FormatJSON(byteCopy), FormatHeaders(req.Header))
	return
}

func (a *Actor) signedHTTPGet(address string) (string, error) {
	req, err := http.NewRequest("GET", address, nil)
	if err != nil {
		log.Error("cannot create new http.request")
		return "", err
	}

	iri, err := url.Parse(address)
	if err != nil {
		log.Error("cannot parse url for GET, check your syntax")
		return "", err
	}

	req.Header.Add("Accept-Charset", "utf-8")
	req.Header.Add("Date", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	req.Header.Add("User-Agent", fmt.Sprintf("%s %s %s", userAgent, libName, version))
	req.Header.Add("host", iri.Host)
	req.Header.Add("digest", "")
	req.Header.Add("Accept", "application/activity+json; profile=\"https://www.w3.org/ns/activitystreams\"")

	// set up the http signer
	signer, _, _ := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, []string{"(request-target)", "date", "host", "digest"}, httpsig.Signature)
	err = signer.SignRequest(a.privateKey, a.publicKeyID, req)
	if err != nil {
		log.Error("Can't sign the request")
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Cannot perform the GET request")
		log.Error(err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {

		responseData, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("GET request to %s failed (%d): %s \n%s", iri.String(), resp.StatusCode, resp.Status, FormatJSON(responseData))
	}

	responseData, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("GET request succeeded:", iri.String(), req.Header, resp.StatusCode, resp.Status, "\n", FormatJSON(responseData))

	responseText := string(responseData)
	return responseText, nil
}

// NewFollower records a new follower to the actor file
func (a *Actor) NewFollower(iri string, inbox string) error {
	a.followers[iri] = inbox
	return a.save()
}

func (a *Actor) appendToOutbox(iri string) (err error) {
	// create outbox file if it doesn't exist
	var outbox *os.File

	outboxFilePath := storage + slash + "actors" + slash + a.name + slash + "outbox.txt"
	outbox, err = os.OpenFile(outboxFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Info("Cannot create or open outbox file")
		log.Info(err)
		return err
	}
	defer outbox.Close()

	outbox.Write([]byte(iri + "\n"))

	return nil
}

func (a *Actor) batchSend(activity map[string]interface{}, recipients []string) (err error) {
	for _, v := range recipients {
		err := a.signedHTTPPost(activity, v)
		if err != nil {
			log.Info("Failed to deliver message to " + v)
		}
	}
	return
}

func (a *Actor) sendToFollowers(activity map[string]interface{}) (err error) {
	recipients := make([]string, len(a.followers))

	i := 0
	for _, inbox := range a.followers {
		recipients[i] = inbox.(string)
		i++
	}
	a.batchSend(activity, recipients)
	return
}

// Follow a remote user by their iri
func (a *Actor) Follow(user string) (err error) {
	remote, err := NewRemoteActor(user)
	if err != nil {
		log.Info("Can't contact " + user + " to get their inbox")
		return
	}

	follow := make(map[string]interface{})
	hash, id := a.newItemID()

	follow["@context"] = context()
	follow["actor"] = a.iri
	follow["id"] = id
	follow["object"] = user
	follow["type"] = "Follow"

	// if we are not already following them
	if _, ok := a.following[user]; !ok {
		// if we have not been rejected previously
		if _, ok := a.rejected[user]; !ok {
			go func() {
				err := a.signedHTTPPost(follow, remote.inbox)
				if err != nil {
					log.Info("Couldn't follow " + user)
					log.Info(err)
					return
				}
				// save the activity
				a.saveItem(hash, follow)
				// we are going to save only on accept so look at
				// the http handler for the accept code
			}()
		}
	}

	return nil
}

// Unfollow the user declared by the iri in `user`
// this recreates the original follow activity
// , wraps it in an Undo activity, sets it's
// id to the id of the original Follow activity that
// was accepted when initially following that user
// (this is read from the `actor.following` map
func (a *Actor) Unfollow(user string) {
	log.Info("Unfollowing " + user)

	// create an undo activiy
	undo := make(map[string]interface{})
	undo["@context"] = context()
	undo["actor"] = a.iri

	// find the id of the original follow
	hash := a.following[user].(string)

	follow := make(map[string]interface{})

	follow["@context"] = context()
	follow["actor"] = a.iri
	follow["id"] = baseURL + "/item/" + hash
	follow["object"] = user
	follow["type"] = "Follow"

	// add the properties to the undo activity
	undo["object"] = follow

	// get the remote user's inbox
	remoteUser, err := NewRemoteActor(user)
	if err != nil {
		log.Info("Failed to contact remote actor")
		return
	}

	// only if we're already following them
	if _, ok := a.following[user]; ok {
		PrettyPrint(undo)
		go func() {
			err := a.signedHTTPPost(undo, remoteUser.inbox)
			if err != nil {
				log.Info("Couldn't unfollow " + user)
				log.Info(err)
				return
			}
			// if there was no error then delete the follow
			// from the list
			delete(a.following, user)
			a.save()
		}()
	}
}

// Announce this activity to our followers
func (a *Actor) Announce(url string) {
	// our announcements are public. Public stuff have a "To" to the url below
	toURL := "https://www.w3.org/ns/activitystreams#Public"
	hash, id := a.newItemID()

	announce := make(map[string]interface{})

	announce["@context"] = context()
	announce["id"] = id
	announce["type"] = "Announce"
	announce["object"] = url
	announce["actor"] = a.iri
	announce["to"] = toURL

	// cc this to all our followers one by one
	// I've seen activities to just include the url of the
	// collection but for now this works.

	// It seems that sharedInbox will be deprecated
	// so this is probably a better idea anyway (#APConf)
	announce["cc"] = a.followersSlice()

	// add a timestamp
	announce["published"] = time.Now().Format(time.RFC3339)

	a.appendToOutbox(announce["id"].(string))
	a.saveItem(hash, announce)
	a.sendToFollowers(announce)
}

func (a *Actor) followersSlice() []string {
	followersSlice := make([]string, len(a.followers))
	for k := range a.followers {
		followersSlice = append(followersSlice, k)
	}
	return followersSlice
}

// Accept a follow request
func (a *Actor) Accept(follow map[string]interface{}) {
	// it's a follow, write it down
	newFollower := follow["actor"].(string)
	// check we aren't following ourselves
	if newFollower == follow["object"] {
		log.Info("You can't follow yourself")
		return
	}

	follower, err := NewRemoteActor(follow["actor"].(string))

	// check if this user is already following us
	if _, ok := a.followers[newFollower]; ok {
		log.Info("You're already following us, yay!")
		// do nothing, they're already following us
	} else {
		a.NewFollower(newFollower, follower.inbox)
	}
	// send accept anyway even if they are following us already
	// this is very verbose. I would prefer creating a map by hand

	// remove @context from the inner activity
	delete(follow, "@context")

	accept := make(map[string]interface{})

	accept["@context"] = "https://www.w3.org/ns/activitystreams"
	accept["to"] = follow["actor"]
	accept["id"], _ = a.newID()
	accept["actor"] = a.iri
	accept["object"] = follow
	accept["type"] = "Accept"

	if err != nil {
		log.Info("Couldn't retrieve remote actor info, maybe server is down?")
		log.Info(err)
	}

	// Maybe we need to save this accept?
	go a.signedHTTPPost(accept, follower.inbox)

}
