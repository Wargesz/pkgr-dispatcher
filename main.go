package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var blueprints chan Blueprint
var conns map[*websocket.Conn]Blueprint

type Blueprint struct {
	Name   string   `json:"name"`
	Source string   `json:"source"`
	Steps  []string `json:"steps"`
}

func main() {
	conns = make(map[*websocket.Conn]Blueprint)
	blueprints = make(chan Blueprint, 5)
	http.HandleFunc("/", sayHi)
	http.HandleFunc("/build", build)
	http.HandleFunc("/order", order)
	http.HandleFunc("/status", status)
	log.Print("serving")
	http.ListenAndServe(":8808", nil)
}

func sayHi(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello")
}

func status(w http.ResponseWriter, r *http.Request) {
	for k := range conns {
		io.WriteString(w, k.NetConn().RemoteAddr().String()+": "+conns[k].Name)
	}
}

func order(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile("blueprints/" + r.URL.Query().Get("bp") + ".json")
	if err != nil {
		log.Panic("file read:", err)
	}
	var bp Blueprint
	json.Unmarshal(content, &bp)
	blueprints <- bp
}

func build(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Panic("read:", err)
		}
		log.Print(string(msg))
		conns[c] = Blueprint{}
		task := <-blueprints
		conns[c] = task
		err = c.WriteJSON(task)
		fmt.Println(task)
		if err != nil {
			log.Println("write:", err)
			break
		}
		_, _, err = c.ReadMessage()
		if err != nil {
			log.Panic("read:", err)
		}
		delete(conns, c)
	}
}
