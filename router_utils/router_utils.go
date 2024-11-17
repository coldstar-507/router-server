package router_utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type ROUTER_TYPE = string
type SERVER_NUMBER = string

type RouterImpl struct {
	Port    int                           `json:"port"`
	Servers map[SERVER_NUMBER]*ServerImpl `json:"servers"`
}

type MetaRouter = map[ROUTER_TYPE]*RouterImpl

func HexToUint16(hex string) uint16 {
	i, _ := strconv.ParseInt(hex, 0, 0)
	return uint16(i)
}

func Uint16ToHex(i uint16) string {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, i)
	return "0x" + hex.EncodeToString(b)
}

const (
	NODE_ROUTER  ROUTER_TYPE = "Node_Router"
	CHAT_ROUTER  ROUTER_TYPE = "Chat_Router"
	MEDIA_ROUTER ROUTER_TYPE = "Media_Router"
)

type ServerImpl struct {
	RouterType ROUTER_TYPE     `json:"serverType"`
	Place      SERVER_NUMBER   `json:"place"`
	IP         string          `json:"ip"`
	RelMedias  []SERVER_NUMBER `json:"relMedias"`
	RelNodes   []SERVER_NUMBER `json:"relNodes"`
	RelChats   []SERVER_NUMBER `json:"relChats"`
}

func InitLocalServer(ip string, place SERVER_NUMBER, routerType ROUTER_TYPE) {
	LocalServer = &ServerImpl{
		IP:         ip,
		RouterType: routerType,
		Place:      place,
		RelMedias:  make([]SERVER_NUMBER, 0, 10),
		RelNodes:   make([]SERVER_NUMBER, 0, 10),
		RelChats:   make([]SERVER_NUMBER, 0, 10),
	}
}

var LocalServer Server

var localMetaRouter MetaRouter

func NodeRouter() Router {
	return localMetaRouter[NODE_ROUTER]
}

func MediaRouter() Router {
	return localMetaRouter[MEDIA_ROUTER]
}

func ChatRouter() Router {
	return localMetaRouter[CHAT_ROUTER]
}

func HandlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong\n"))
}

func HandleServerStatus(w http.ResponseWriter, r *http.Request) {
	if b, err := json.MarshalIndent(LocalServer, "", "    "); err != nil {
		panic(err)
	} else {
		w.Write(b)
	}
}

func HandleRouterStatus(w http.ResponseWriter, r *http.Request) {
	if b, err := json.MarshalIndent(localMetaRouter, "", "    "); err != nil {
		panic(err)
	} else {
		w.Write(b)
	}
}

func HandleScoreRequest(w http.ResponseWriter, r *http.Request) {
	if b, err := json.Marshal(LocalServer.Scores()); err != nil {
		log.Println("HandleScoreRequest error encoding score:", err)
		w.WriteHeader(500)
	} else if _, err := w.Write(b); err != nil {
		log.Println("HandleScoreRequest error writing score:", err)
		w.WriteHeader(501)
	}
}

type Router interface {
	Host(place SERVER_NUMBER) string
	GetPort() int
	HostAndPort(place SERVER_NUMBER) string
	GetServer(place SERVER_NUMBER) Server
	RelativeMedias(place SERVER_NUMBER) []SERVER_NUMBER
	RelativeNodes(place SERVER_NUMBER) []SERVER_NUMBER
	RelativeChats(place SERVER_NUMBER) []SERVER_NUMBER
}

type Server interface {
	Run()
	Scores() *Scores
	RelativeMedias() []SERVER_NUMBER
	RelativeNodes() []SERVER_NUMBER
	RelativeChats() []SERVER_NUMBER
}

func (lr *ServerImpl) Scores() *Scores {
	return &Scores{
		Medias: lr.RelMedias,
		Nodes:  lr.RelNodes,
		Chats:  lr.RelChats,
	}
}

func SetMetaRouter(m MetaRouter) {
	localMetaRouter = m
}

func FetchMetaRouter() MetaRouter {
	url := "http://localhost:8084/full-router"
	r, err := http.DefaultClient.Get(url)
	if err != nil {
		log.Println("FetchMetaRouter(): error fetching meta router:", err)
	}
	defer r.Body.Close()
	var m MetaRouter
	if err = json.NewDecoder(r.Body).Decode(&m); err != nil {
		log.Println("FetchMetaRouter(): error decoding meta router:", err)
	}
	return m
}

func (lr *ServerImpl) fetchMetaRouter() {
	url := "http://localhost:8084/full-router"
	r, err := http.DefaultClient.Get(url)
	if err != nil {
		log.Printf("Server %s@%s error fetching meta router: %v\n",
			lr.IP, lr.Place, err)
	}
	defer r.Body.Close()
	var m MetaRouter
	if err = json.NewDecoder(r.Body).Decode(&m); err != nil {
		log.Printf("Server %s@%s error decoding meta router: %v\n",
			lr.IP, lr.Place, err)
	}
	localMetaRouter = m
}

func (lr *ServerImpl) pushServer() {
	b, err := json.Marshal(lr)
	if err != nil {
		err = fmt.Errorf("Server %s@%s error marshalling server: %v",
			lr.IP, lr.Place, err)
		panic(err)
	}

	url := "http://localhost:8084/new-server"
	r, err := http.DefaultClient.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		err = fmt.Errorf("Server %s@%s error pushing to meta router: %v",
			lr.IP, lr.Place, err)
		panic(err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		err = fmt.Errorf("Server %s@%s status code is not 200: %d",
			lr.IP, lr.Place, r.StatusCode)
		panic(err)
	}

}

func (lr *ServerImpl) Run() {
	tck1 := time.NewTicker(time.Minute * 1)
	// tck2 := time.NewTicker(time.Minute * 2)

	calcChats := func() {
		log.Println("LocalRouter: calculating chat routes")
		lr.RelChats = calculateRoutes(localMetaRouter[CHAT_ROUTER])
	}
	calcNodes := func() {
		log.Println("LocalRouter: calculating node routes")
		lr.RelNodes = calculateRoutes(localMetaRouter[NODE_ROUTER])
	}
	calcMedias := func() {
		log.Println("LocalRouter: calculating media routes")
		lr.RelMedias = calculateRoutes(localMetaRouter[MEDIA_ROUTER])
	}

	lr.pushServer()
	for {
		log.Println("LocalRouter: periodic routes calculation")
		lr.fetchMetaRouter()
		go calcChats()
		go calcNodes()
		go calcMedias()
		<-tck1.C
	}

}

func calculateRoutes(r *RouterImpl) []SERVER_NUMBER {
	type score struct {
		id    SERVER_NUMBER
		score int64
	}

	var futureScores = make(map[SERVER_NUMBER]<-chan *int64)
	for p := range r.Servers {
		futureScores[p] = r.Ping(p)
	}

	scores := make([]*score, 0, len(r.Servers))
	for x, v := range futureScores {
		if s := <-v; s != nil {
			scores = append(scores, &score{id: x, score: *s})
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score < scores[j].score
	})

	res := make([]SERVER_NUMBER, len(scores))
	for i, x := range scores {
		res[i] = x.id
	}

	return res
}

func (r *RouterImpl) Host(place SERVER_NUMBER) string {
	return r.Servers[place].IP
}

func (r *RouterImpl) GetPort() int {
	return r.Port
}

func (r *RouterImpl) HostAndPort(place SERVER_NUMBER) string {
	return r.Servers[place].IP + ":" + strconv.Itoa(r.Port)
}

func (r *RouterImpl) GetServer(place SERVER_NUMBER) Server {
	return r.Servers[place]
}

func (r *RouterImpl) RelativeMedias(place SERVER_NUMBER) []SERVER_NUMBER {
	return r.GetServer(place).RelativeMedias()

}

func (r *RouterImpl) RelativeNodes(place SERVER_NUMBER) []SERVER_NUMBER {
	return r.GetServer(place).RelativeNodes()
}

func (r *RouterImpl) RelativeChats(place SERVER_NUMBER) []SERVER_NUMBER {
	return r.GetServer(place).RelativeChats()
}

func (s *ServerImpl) RelativeMedias() []SERVER_NUMBER {
	return s.RelMedias
}

func (s *ServerImpl) RelativeNodes() []SERVER_NUMBER {
	return s.RelNodes
}

func (s *ServerImpl) RelativeChats() []SERVER_NUMBER {
	return s.RelChats
}

func (r *RouterImpl) Ping(place SERVER_NUMBER) <-chan *int64 {
	ch := make(chan *int64)
	go func() {
		url := "http://" + r.HostAndPort(place) + "/ping"
		t1 := time.Now()
		if res, err := http.DefaultClient.Get(url); err != nil {
			log.Printf("Ping(%s) error creating req: %v", place, err)
			ch <- nil
		} else if res.StatusCode != 200 {
			log.Printf("Ping(%s) error: code isn't 200: %v", place, err)
			ch <- nil
		} else {
			ms := time.Since(t1).Milliseconds()
			log.Printf("Ping(%s) took %d ms", place, ms)
			ch <- &ms
		}
		close(ch)
	}()
	return ch
}

type Scores struct {
	Medias []SERVER_NUMBER `json:"mediaPlaces"`
	Nodes  []SERVER_NUMBER `json:"nodePlaces"`
	Chats  []SERVER_NUMBER `json:"chatPlaces"`
}

func (r *RouterImpl) fetchScores(place SERVER_NUMBER) <-chan *Scores {
	addr := "http://" + r.HostAndPort(place) + "/route-scores"
	ch := make(chan *Scores)
	go func() {
		var sc Scores
		if req, err := http.NewRequest("GET", addr, nil); err != nil {
			log.Println("fetchScores error making req:", err)
			ch <- nil
		} else if res, err := http.DefaultClient.Do(req); err != nil {
			log.Println("fetchScores error doing req:", err)
			ch <- nil
		} else if res.StatusCode != 200 {
			log.Println("fetchScores status != 200")
			ch <- nil
		} else if err = json.NewDecoder(res.Body).Decode(&sc); err != nil {
			log.Println("fetchScores error decoding res json:", err)
			ch <- nil
		} else {
			ch <- &sc
		}
		close(ch)
	}()
	return ch
}
