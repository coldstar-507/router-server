package the_router

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/coldstar-507/router/router_utils"
)

func HandleNewServer(w http.ResponseWriter, r *http.Request) {
	var sr router_utils.ServerImpl
	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		log.Println("HandleNewServer(): error:", err)
		w.WriteHeader(500)
	} else {
		TheMetaRouter[sr.RouterType].Servers[sr.Place] = &sr
	}
}

func HandleFullRouter(w http.ResponseWriter, r *http.Request) {
	if b, err := json.MarshalIndent(TheMetaRouter, "", "    "); err != nil {
		log.Println("HandleFullRouter(): error marshalling metaRouter:", err)
		w.WriteHeader(500)
	} else {
		w.Write(b)
	}
}

var TheMetaRouter = router_utils.MetaRouter{
	router_utils.NODE_ROUTER: &router_utils.RouterImpl{
		Port:    8083,
		Servers: map[router_utils.SERVER_NUMBER]*router_utils.ServerImpl{},
	},

	router_utils.MEDIA_ROUTER: &router_utils.RouterImpl{
		Port:    8081,
		Servers: map[router_utils.SERVER_NUMBER]*router_utils.ServerImpl{},
	},

	router_utils.CHAT_ROUTER: &router_utils.RouterImpl{
		Port:    8082,
		Servers: map[router_utils.SERVER_NUMBER]*router_utils.ServerImpl{},
	},
}
