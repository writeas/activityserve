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
	name, summary, actorType, iri string
	followersIRI                  string
	nuIri                         *url.URL
	followers, following          map[string]interface{}
	posts                         map[int]map[string]interface{}
	publicKey                     crypto.PublicKey
	privateKey                    crypto.PrivateKey
	publicKeyPem                  string
	privateKeyPem                 string
	publicKeyID                   string
}

// ActorToSave is a stripped down actor representation
// with exported properties in order for json to be
// able to marshal it.
// see https://stackoverflow.com/questions/26327391/json-marshalstruct-returns
type ActorToSave struct {
	Name, Summary, ActorType, IRI, PublicKey, PrivateKey string
	Followers, Following                                 map[string]interface{}
}

// MakeActor returns a new local actor we can act
// on behalf of
func MakeActor(name, summary, actorType string) (Actor, error) {
	followers := make(map[string]interface{})
	following := make(map[string]interface{})
	followersIRI := baseURL + name + "/followers"
	publicKeyID := baseURL + name + "#main-key"
	iri := baseURL + "/" + name
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
		followersIRI: followersIRI,
		publicKeyID:  publicKeyID,
	}

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
		publicKey:     publicKey,
		privateKey:    privateKey,
		publicKeyPem:  jsonData["PublicKey"].(string),
		privateKeyPem: jsonData["PrivateKey"].(string),
		followersIRI:  baseURL + name + "/followers",
		publicKeyID:   baseURL + name + "#main-key",
	}

	return actor, nil
}

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
	"followers": "` + baseURL + a.name + `/followers",
	"following": "` + baseURL + a.name + `/following",
	"liked": "` + baseURL + a.name + `/liked",
	"publicKey": {
		"id": "` + baseURL + a.name + `#main-key",
		"owner": "` + baseURL + a.name + `",
		"publicKeyPem": "` + strings.ReplaceAll(a.publicKeyPem, "\n", "\\n") + `"
	  }
	}`
}

func (a *Actor) newID() string {
	return uniuri.New()
}

// CreateNote posts an activityPub note to our followers
func (a *Actor) CreateNote(content string) {
	// for now I will just write this to the outbox

	id := a.newID()
	create := make(map[string]interface{})
	note := make(map[string]interface{})
	context := make([]string, 1)
	context[0] = "https://www.w3.org/ns/activitystreams"
	create["@context"] = context
	create["actor"] = baseURL + a.name
	create["cc"] = a.followersIRI
	create["id"] = baseURL + a.name + "/" + id
	create["object"] = note
	note["attributedTo"] = baseURL + a.name
	note["cc"] = a.followersIRI
	note["content"] = content
	note["inReplyTo"] = "https://cybre.space/@qwazix/102688373602724023"
	note["id"] = baseURL + a.name + "/note/" + id
	note["published"] = time.Now().Format(time.RFC3339)
	note["url"] = create["id"]
	note["type"] = "Note"
	note["to"] = "https://www.w3.org/ns/activitystreams#Public"
	create["published"] = note["published"]
	create["type"] = "Create"

	// note := `{
	// 	"actor" : "https://` + baseURL + a.name + `",
	// 	"cc" : [
	// 	   "https://` + baseURL + a.name + `/followers"
	// 	],
	// 	"id" : "https://` + baseURL + a.name + `/` + id +`",
	// 	"object" : {
	// 	   "attributedTo" : "https://` + baseURL + a.name + `",
	// 	   "cc" : [
	// 		  "https://` + baseURL + a.name + `/followers"
	// 	   ],
	// 	   "content" : "`+ content + `",
	// 	   "id" : "https://` + baseURL + a.name + `/` + id +`",
	// 	   "inReplyTo" : null,
	// 	   "published" : "2019-08-26T16:25:26Z",
	// 	   "to" : [
	// 		  "https://www.w3.org/ns/activitystreams#Public"
	// 	   ],
	// 	   "type" : "Note",
	// 	   "url" : "https://` + baseURL + a.name + `/` + id +`"
	// 	},
	// 	"published" : "2019-08-26T16:25:26Z",
	// 	"to" : [
	// 	   "https://www.w3.org/ns/activitystreams#Public"
	// 	],
	// 	"type" : "Create"
	//  }`
	to, _ := url.Parse("https://cybre.space/inbox")
	go a.send(create, to)
	a.saveItem(id, create)
}

func (a *Actor) saveItem(id string, content map[string]interface{}) error {
	JSON, _ := json.MarshalIndent(content, "", "\t")

	dir := storage + slash + "actors" + slash + a.name + slash + "items"
	err := ioutil.WriteFile(dir+slash+id+".json", JSON, 0644)
	if err != nil {
		log.Printf("WriteFileJson ERROR: %+v", err)
		return err
	}
	return nil
}

// send is here for backward compatibility and maybe extra pre-processing
// not always required
func (a *Actor) send(content map[string]interface{}, to *url.URL) (err error) {
	return a.signedHTTPPost(content, to.String())
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
	req.Header.Add("User-Agent", fmt.Sprintf("activityserve 0.0"))
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
		err = fmt.Errorf("POST request to %s failed (%d): %s\nResponse: %s \nRequest: %s \nHeaders: %s", to, resp.StatusCode, resp.Status, formatJSON(responseData), formatJSON(byteCopy), req.Header)
		log.Info(err)
		return
	}
	responseData, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("POST request to %s succeeded (%d): %s \nResponse: %s \nRequest: %s \nHeaders: %s", to, resp.StatusCode, resp.Status, formatJSON(responseData), formatJSON(byteCopy), req.Header)
	return
}

func (a *Actor) signedHTTPGet(address string) (string, error){
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
		return "", fmt.Errorf("GET request to %s failed (%d): %s \n%s", iri.String(), resp.StatusCode, resp.Status, formatJSON(responseData))
	}

	responseData, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("GET request succeeded:", iri.String(), req.Header, resp.StatusCode, resp.Status, "\n", formatJSON(responseData))

	responseText := string(responseData)
	return responseText, nil
}