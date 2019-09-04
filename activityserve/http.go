package activityserve

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gologme/log"
	"github.com/gorilla/mux"

	"encoding/json"
)

// SetupHTTP starts an http server with all the required handlers
func Serve() {

	var webfingerHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/jrd+json; charset=utf-8")
		account := r.URL.Query().Get("resource")              // should be something like acct:user@example.com
		account = strings.Replace(account, "acct:", "", 1)    // remove acct:
		server := strings.Split(baseURL, "://")[1]            // remove protocol from baseURL. Should get example.com
		server = strings.TrimSuffix(server, "/")              // remove protocol from baseURL. Should get example.com
		account = strings.Replace(account, "@"+server, "", 1) // remove server from handle. Should get user
		actor, err := LoadActor(account)
		// error out if this actor does not exist
		if err != nil {
			log.Info("No such actor")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404 - actor not found")
			return
		}
		// response := `{"subject":"acct:` + actor.name + `@` + server + `","aliases":["` + baseURL + actor.name + `","` + baseURL + actor.name + `"],"links":[{"href":"` + baseURL + `","type":"text/html","rel":"https://webfinger.net/rel/profile-page"},{"href":"` + baseURL + actor.name + `","type":"application/activity+json","rel":"self"}]}`

		responseMap := make(map[string]interface{})

		responseMap["subject"] = "acct:" + actor.name + "@" + server
		links := make(map[string]string)
		links["rel"] = "self"
		links["type"] = "application/activity+json"
		links["href"] = baseURL + actor.name
		responseMap["links"] = links

		response, err := json.Marshal(responseMap)
		if err != nil {
			log.Error("problem creating the webfinger response json")
		}
		log.Info(string(response))
		w.Write([]byte(response))
	}

	var actorHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		log.Info("Remote server just fetched our /actor endpoint")
		username := mux.Vars(r)["actor"]
		log.Info(username)
		if username == ".well-known" || username == "favicon.ico" {
			log.Info("well-known, skipping...")
			return
		}
		actor, err := LoadActor(username)
		// error out if this actor does not exist (or there are dots or slashes in his name)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404 - page not found")
			log.Info("Can't create local actor")
			return
		}
		fmt.Fprintf(w, actor.whoAmI())
		log.Info(r.RemoteAddr)
		log.Info(r.Body)
		log.Info(r.Header)
	}

	var outboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		username := mux.Vars(r)["actor"]  // get the needed actor from the muxer (url variable {actor} below)
		actor, err := LoadActor(username) // load the actor from disk
		if err != nil {                   // either actor requested has illegal characters or
			log.Info("Can't load local actor") // we don't have such actor
			fmt.Fprintf(w, "404 - page not found")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var response string
		if r.URL.Query().Get("page") == "" {
			//TODO fix total items
			response = `{
				"@context" : "https://www.w3.org/ns/activitystreams",
				"first" : "` + baseURL + actor.name + `/outbox?page=true",
				"id" : "` + baseURL + actor.name + `/outbox",
				"last" : "` + baseURL + actor.name + `/outbox?min_id=0&page=true",
				"totalItems" : 10, 
				"type" : "OrderedCollection"
			 }`
		} else {
			content := "Hello, World!"
			id := "asfdasdf"
			response = `
		{
			"@context" : "https://www.w3.org/ns/activitystreams",
			"id" : "` + baseURL + actor.name + `/outbox?min_id=0&page=true",
			"next" : "` + baseURL + actor.name + `/outbox?max_id=99524642494530460&page=true",
			"orderedItems" :[
				{
					"actor" : "https://` + baseURL + actor.name + `",
					"cc" : [
					   "https://` + baseURL + actor.name + `/followers"
					],
					"id" : "https://` + baseURL + actor.name + `/` + id + `",
					"object" : {
					   "attributedTo" : "https://` + baseURL + actor.name + `",
					   "cc" : [
						  "https://` + baseURL + actor.name + `/followers"
					   ],
					   "content" : "` + content + `",
					   "id" : "https://` + baseURL + actor.name + `/` + id + `",
					   "inReplyTo" : null,
					   "published" : "2019-08-26T16:25:26Z",
					   "to" : [
						  "https://www.w3.org/ns/activitystreams#Public"
					   ],
					   "type" : "Note",
					   "url" : "https://` + baseURL + actor.name + `/` + id + `"
					},
					"published" : "2019-08-26T16:25:26Z",
					"to" : [
					   "https://www.w3.org/ns/activitystreams#Public"
					],
					"type" : "Create"
				 }
			],
			"partOf" : "` + baseURL + actor.name + `/outbox",
			"prev" : "` + baseURL + actor.name + `/outbox?min_id=99982453036184436&page=true",
			"type" : "OrderedCollectionPage"
		 }`
		}
		w.Write([]byte(response))
	}

	// Add the handlers to a HTTP server
	gorilla := mux.NewRouter()
	gorilla.HandleFunc("/.well-known/webfinger", webfingerHandler)
	gorilla.HandleFunc("/{actor}/outbox", outboxHandler)
	gorilla.HandleFunc("/{actor}/outbox/", outboxHandler)
	// gorilla.HandleFunc("/{actor}/inbox", inboxHandler)
	// gorilla.HandleFunc("/{actor}/inbox/", inboxHandler)
	gorilla.HandleFunc("/{actor}/", actorHandler)
	gorilla.HandleFunc("/{actor}", actorHandler)
	// gorilla.HandleFunc("/{actor}/post/{hash}", postHandler)
	http.Handle("/", gorilla)

	log.Fatal(http.ListenAndServe(":8081", nil))
}
