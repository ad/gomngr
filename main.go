package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kardianos/osext"

	"./selfupdate"
	"github.com/gorilla/websocket"
	"github.com/nu7hatch/gouuid"
)

const version = "0.0.1"

var mu, _ = uuid.NewV4()
var addr = flag.String("addr", "localhost:80", "cc address:port")
var mngruuid = flag.String("uuid", mu.String(), "manager uuid")

type Action struct {
	ZondUUID   string `json:"zond"`
	MngrUUID   string `json:"manager"`
	Creator    string `json:"creator"`
	Type       string `json:"type"`
	Action     string `json:"action"`
	Param      string `json:"param"`
	Result     string `json:"result"`
	ParentUUID string `json:"parent"`
	Created    int64  `json:"created"`
	Updated    int64  `json:"updated"`
	Target     string `json:"target"`
	Repeat     string `json:"repeat"`
	UUID       string `json:"uuid"`
}

func main() {
	log.Printf("Started version %s", version)

	go selfupdate.StartSelfupdate("ad/gomngr", version)

	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/sub/mngrtasks,mngr" + *mngruuid}
	log.Printf("connecting to %s", u.String())

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), http.Header{"X-MngrUuid": {*mngruuid}})
	if ws != nil {
		defer ws.Close()
	}
	if err != nil {
		log.Fatal("dial:", err)
	}
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := ws.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
				restart()
			}
			// log.Printf("recv: %s", message)
			var action = new(Action)
			err = json.Unmarshal(message, &action)
			if err != nil {
				fmt.Println("error:", err)
			} else {
				if action.Action != "alive" {
					fmt.Printf("%+v\n", action)
				}

				if action.Type == "measurement" {
					// TODO:
					// Block task
					// Find the correct number of zonds with the same destination parameter as in main task
					// Create a subtask for each zond (+ set uuid of the main task)
					// Send posts to pubsub with task metadata
					// Wait for a while (timeout/deadline from the main task)
					// Delete / Hide / Mark Unfinished Jobs
					// Make a calculation with data from the completed tasks
					// Write the result to the main task
				} else if action.Action == "alive" {
					ccAddr := *addr
					action.MngrUUID = *mngruuid
					js, _ := json.Marshal(action)
					post("http://"+ccAddr+"/mngr/pong", string(js))
				}
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")
			err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func restart() {
	file, err := osext.Executable()
	if err != nil {
		log.Println("restart:", err)
	} else {
		err = syscall.Exec(file, os.Args, os.Environ())
		if err != nil {
			log.Fatal(err)
		}
	}
}

func post(url string, jsonData string) string {
	var jsonStr = []byte(jsonData)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MngrUuid", *mngruuid)

	client := &http.Client{}
	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Println(err)
		return "error"
	} else {
		if resp.StatusCode == 429 {
			log.Printf("%s: %d", url, resp.StatusCode)
			time.Sleep(time.Duration(rand.Intn(30)) * time.Second)
			return post(url, jsonData)
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			return string(body)
		}
	}
}
