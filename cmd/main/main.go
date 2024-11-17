package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/coldstar-507/router/internal/the_router"
	"github.com/coldstar-507/router/router_utils"
	"github.com/coldstar-507/utils/http_utils"
	"github.com/coldstar-507/utils/utils"
)

func updateServerScores(r *router_utils.RouterImpl, s *router_utils.ServerImpl) {
	h := r.HostAndPort(s.Place)
	url := "http://" + h + "/route-scores"
	res, err := http.DefaultClient.Get(url)
	if err != nil {
		log.Println("MetaRouter periodic scores err:", err)
	} else if res.StatusCode != 200 {
		log.Println("MetaRouter periodic scores code:", res.StatusCode)
	} else {
		var sc router_utils.Scores
		err = json.NewDecoder(res.Body).Decode(&sc)
		if err != nil {
			log.Println("MetaRouter periodic scores code:", res.StatusCode)
		} else {
			s.RelChats = sc.Chats
			s.RelNodes = sc.Nodes
			s.RelMedias = sc.Medias
		}
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /new-server", the_router.HandleNewServer)
	mux.HandleFunc("GET /full-router", the_router.HandleFullRouter)
	mux.HandleFunc("GET /ping", router_utils.HandlePing)

	server := http_utils.ApplyMiddlewares(mux, http_utils.StatusLogger)

	go func() {
		tck := time.NewTicker(time.Second * 70)
		for {
			<-tck.C
			b, _ := json.MarshalIndent(the_router.TheMetaRouter, "", "    ")
			log.Println("MetaRouter periodic log:\n", string(b))
		}
	}()

	go func() {
		tck := time.NewTicker(time.Minute * 1)
		for {
			<-tck.C
			log.Println("MetaRouter periodic server scores udpate")
			for _, r := range the_router.TheMetaRouter {
				for _, s := range r.Servers {
					go updateServerScores(r, s)
				}
			}

		}
	}()

	addr := "0.0.0.0:8084"
	log.Println("Starting http router-server on", addr)
	err := http.ListenAndServe(addr, server)
	utils.NonFatal(err, "http.ListenAndServe error")

}
