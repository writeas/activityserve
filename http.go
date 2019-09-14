package activityserve

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gologme/log"
	"github.com/gorilla/mux"

	"encoding/json"
)

// Serve starts an http server with all the required handlers
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
		// links is a json array with a single element
		var links [1]map[string]string
		link1 := make(map[string]string)
		link1["rel"] = "self"
		link1["type"] = "application/activity+json"
		link1["href"] = baseURL + actor.name
		links[0] = link1
		responseMap["links"] = links

		response, err := json.Marshal(responseMap)
		if err != nil {
			log.Error("problem creating the webfinger response json")
		}
		PrettyPrintJSON(response)
		w.Write([]byte(response))
	}

	var actorHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		log.Info("Remote server " + r.RemoteAddr + " just fetched our /actor endpoint")
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

		// Show some debugging information
		printer.Info("")
		body, _ := ioutil.ReadAll(r.Body)
		PrettyPrintJSON(body)
		log.Info(FormatHeaders(r.Header))
		printer.Info("")
	}

	var outboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		pageStr := r.URL.Query().Get("page") // get the page from the query string as string
		username := mux.Vars(r)["actor"]     // get the needed actor from the muxer (url variable {actor} below)
		actor, err := LoadActor(username)    // load the actor from disk
		if err != nil {                      // either actor requested has illegal characters or
			log.Info("Can't load local actor") // we don't have such actor
			fmt.Fprintf(w, "404 - page not found")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		postsPerPage := 100
		var response []byte
		filename := storage + slash + "actors" + slash + actor.name + slash + "outbox.txt"
		totalLines, err := lineCounter(filename)
		if err != nil {
			log.Info("Can't read outbox.txt")
			log.Info(err)
			return
		}
		if pageStr == "" {
			//TODO fix total items
			response = []byte(`{
				"@context" : "https://www.w3.org/ns/activitystreams",
				"first" : "` + baseURL + actor.name + `/outbox?page=1",
				"id" : "` + baseURL + actor.name + `/outbox",
				"last" : "` + baseURL + actor.name + `/outbox?page=` + strconv.Itoa(totalLines/postsPerPage+1) + `",
				"totalItems" : 10, 
				"type" : "OrderedCollection"
			 }`)
		} else {
			page, err := strconv.Atoi(pageStr) // get page number from query string
			if err != nil {
				log.Info("Page number not a number, assuming 1")
				page = 1
			}
			lines, err := ReadLines(filename, (page-1)*postsPerPage, page*(postsPerPage+1)-1)
			if err != nil {
				log.Info("Can't read outbox file")
				log.Info(err)
				return
			}
			responseMap := make(map[string]interface{})
			responseMap["@context"] = context()
			responseMap["id"] = baseURL + actor.name + "/outbox?page=" + pageStr

			if page*postsPerPage < totalLines {
				responseMap["next"] = baseURL + actor.name + "/outbox?page=" + strconv.Itoa(page+1)
			}
			if page > 1 {
				responseMap["prev"] = baseURL + actor.name + "/outbox?page=" + strconv.Itoa(page-1)
			}
			responseMap["partOf"] = baseURL + actor.name + "/outbox"
			responseMap["type"] = "OrderedCollectionPage"

			orderedItems := make([]interface{}, 0, postsPerPage)

			for _, item := range lines {
				// split the line
				parts := strings.Split(item, "/")

				// keep the hash
				hash := parts[len(parts)-1]
				// build the filename
				filename := storage + slash + "actors" + slash + actor.name + slash + "items" + slash + hash + ".json"
				// open the file
				activityJSON, err := ioutil.ReadFile(filename)
				if err != nil {
					log.Error("can't read activity")
					log.Info(filename)
					return
				}
				var temp map[string]interface{}
				// put it into a map
				json.Unmarshal(activityJSON, &temp)
				// append to orderedItems
				orderedItems = append(orderedItems, temp)
			}

			responseMap["orderedItems"] = orderedItems

			response, err = json.Marshal(responseMap)
			if err != nil {
				log.Info("can't marshal map to json")
				log.Info(err)
				return
			}
		}
		w.Write(response)
	}

	var inboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		activity := make(map[string]interface{})
		err = json.Unmarshal(b, &activity)
		if err != nil {
			log.Error("Probably this request didn't have (valid) JSON inside it")
			return
		}
		// TODO check if it's actually an activity

		// check if case is going to be an issue
		switch activity["type"] {
		case "Follow":
			// load the object as actor
			actor, err := LoadActor(mux.Vars(r)["actor"]) // load the actor from disk
			if err != nil {
				log.Error("No such actor")
				return
			}
			actor.OnFollow(activity)
		case "Accept":
			acceptor := activity["actor"].(string)
			actor, err := LoadActor(mux.Vars(r)["actor"]) // load the actor from disk
			if err != nil {
				log.Error("No such actor")
				return
			}

			// From here down this could be moved to Actor (TBD)

			follow := activity["object"].(map[string]interface{})
			id := follow["id"].(string)

			// check if the object of the follow is us
			if follow["actor"].(string) != baseURL+actor.name {
				log.Info("This is not for us, ignoring")
				return
			}
			// try to get the hash only
			hash := strings.Replace(id, baseURL+actor.name+"/", "", 1)
			// if there are still slashes in the result this means the
			// above didn't work
			if strings.ContainsAny(hash, "/") {
				log.Info("The id of this follow is probably wrong")
				return
			}

			// Have we already requested this follow or are we following anybody that
			// sprays accepts?
			savedFollowRequest, err := actor.loadItem(hash)
			if err != nil {
				log.Info("We never requested this follow, ignoring the Accept")
				return
			}
			if savedFollowRequest["id"] != id {
				log.Info("Id mismatch between Follow request and Accept")
				return
			}
			actor.following[acceptor] = hash
			actor.save()
		default:

		}
	}

	var peersHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		username := mux.Vars(r)["actor"]
		collection := mux.Vars(r)["peers"]
		if collection != "followers" && collection != "following" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 - No such collection"))
			return
		}
		actor, err := LoadActor(username)
		// error out if this actor does not exist
		if err != nil {
			log.Info("Can't create local actor")
			return
		}
		var page int
		pageS := r.URL.Query().Get("page")
		if pageS == "" {
			page = 0
		} else {
			page, _ = strconv.Atoi(pageS)
		}
		response, _ := actor.getPeers(page, collection)
		w.Write(response)
	}

	var postHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		username := mux.Vars(r)["actor"]
		hash := mux.Vars(r)["hash"]
		actor, err := LoadActor(username)
		// error out if this actor does not exist
		if err != nil {
			log.Info("Can't create local actor")
			return
		}
		post, err := actor.loadItem(hash)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404 - post not found")
			return
		}
		postJSON, err := json.Marshal(post)
		if err!= nil{
			log.Info("failed to marshal json from item " + hash + " text")
			return
		}
		w.Write(postJSON)
	}

	// Add the handlers to a HTTP server
	gorilla := mux.NewRouter()
	gorilla.HandleFunc("/.well-known/webfinger", webfingerHandler)
	gorilla.HandleFunc("/{actor}/peers/{peers}", peersHandler)
	gorilla.HandleFunc("/{actor}/outbox", outboxHandler)
	gorilla.HandleFunc("/{actor}/outbox/", outboxHandler)
	gorilla.HandleFunc("/{actor}/inbox", inboxHandler)
	gorilla.HandleFunc("/{actor}/inbox/", inboxHandler)
	gorilla.HandleFunc("/{actor}/", actorHandler)
	gorilla.HandleFunc("/{actor}", actorHandler)
	gorilla.HandleFunc("/{actor}/item/{hash}", postHandler)
	http.Handle("/", gorilla)

	log.Fatal(http.ListenAndServe(":8081", nil))
}
