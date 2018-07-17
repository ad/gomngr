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
	"strconv"
	"time"

	"github.com/ad/gocc/ccredis"
	"github.com/ad/gomngr/selfupdate"
	"github.com/ad/gomngr/utils"
	"github.com/gorilla/websocket"
	"github.com/nu7hatch/gouuid"
)

const version = "0.0.5"

var mu, _ = uuid.NewV4()
var addr = flag.String("addr", "localhost:80", "cc address:port")
var mngruuid = flag.String("uuid", mu.String(), "manager uuid")

var results = make(chan string, 1)

type Action struct {
	ZondUUID   string `json:"zond"`
	MngrUUID   string `json:"manager"`
	Creator    string `json:"creator"`
	Type       string `json:"type"`
	Count      int64  `json:"count"`
	TimeOut    int64  `json:"timeout"`
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

type Task struct {
	ZondUUID string `json:"zond"`
	Created  int64  `json:"created"`
	UUID     string `json:"uuid"`
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
				utils.Restart()
			}
			log.Printf("recv: %s", message)
			var action = new(Action)
			err = json.Unmarshal(message, &action)
			if err != nil {
				fmt.Println("error:", err)
			} else {
				if action.Action != "alive" {
					fmt.Printf("%+v\n", action)
				}

				if action.Type == "measurement" {
					if action.Result != "" {
						// send result to results channel
						results <- string(message)
					} else {
						// 1. Block task
						var postAction = Action{MngrUUID: *mngruuid, Action: "block", Result: "", UUID: action.UUID}
						var js, _ = json.Marshal(postAction)
						var status = post("http://"+*addr+"/mngr/task/block", string(js))

						if status != `{"status": "ok", "message": "ok"}` {
							if status != `{"status": "error", "message": "task not found"}` {
								log.Println(action.UUID, status)
							}
						} else {
							// 2. Find the correct number of zonds with the same destination parameter as in main task
							// 3. Create a subtask for each zond (+ set uuid of the main task)
							// 4. Send posts to pubsub with task metadata
							// 5. Wait for a while (timeout/deadline from the main task)
							// 6. Delete / Hide / Mark Unfinished Jobs
							// 7. Make a calculation with data from the completed tasks
							// 8. Write the result to the main task
							go processTask(action)
						}
					}
				} else if action.Action == "alive" {
					ccAddr := *addr
					action.MngrUUID = *mngruuid
					js, _ := json.Marshal(action)
					post("http://"+ccAddr+"/mngr/pong", string(js))
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case res := <-results:
				log.Printf("recv: %s", res)
				var action Action
				err := json.Unmarshal([]byte(res), &action)
				if err != nil {
					log.Println(err.Error())
				}

				taskjson, _ := ccredis.Client.Get("task/" + action.ParentUUID).Result()
				var task Action
				err = json.Unmarshal([]byte(taskjson), &task)
				if err != nil {
					log.Println(err.Error())
				}
				if task.Result != "" {
					// if all subtasks finished — call finishTask
					finishTask(&action)
				} else {
					// tasksCount, _ := client.SCard("tasks/measurement/"+action.ParentUUID).Result()
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
	}

	if resp.StatusCode == 429 {
		log.Printf("%s: %d", url, resp.StatusCode)
		time.Sleep(time.Duration(rand.Intn(30)) * time.Second)
		return post(url, jsonData)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return string(body)
}

func processTask(action *Action) {
	// 2. Find the correct number of zonds with the same destination parameter as in main task
	for i := 0; i < int(action.Count); i++ {
		u, _ := uuid.NewV4()
		var UUID = u.String()
		var msec = time.Now().Unix()

		// 3. Create a subtask for each zond (+ set uuid of the main task)
		newAction := Action{
			Action: action.Action,
			// ZondUUID:   task.ZondUUID,
			MngrUUID:   *mngruuid,
			Creator:    action.Creator,
			Type:       "task",
			Param:      action.Param,
			ParentUUID: action.UUID,
			Created:    msec,
			UUID:       UUID,
			Target:     action.Target,
		}
		js, _ := json.Marshal(newAction)

		ccredis.Client.SAdd("tasks-new", UUID)
		ccredis.Client.SAdd("tasks/measurement/"+action.UUID, UUID)
		ccredis.Client.Set("task/"+UUID, string(js), time.Duration(action.TimeOut+300)*time.Second) // subtask ttl is 5 minutes

		// 4. Send posts to pubsub with task metadata
		go post("http://127.0.0.1:80/pub/"+action.Target, string(js))
	}

	// 5. Wait for a while (timeout/deadline from the main task)
	select {
	case <-time.After(time.Duration(action.TimeOut) * time.Second):
		finishTask(action)
	}
}

func finishTask(action *Action) {
	// check if task already finished
	taskjson, _ := ccredis.Client.Get("task/" + action.UUID).Result()
	var task Action
	err := json.Unmarshal([]byte(taskjson), &task)
	if err != nil {
		log.Println(err.Error())
	}
	if task.Result != "" {
		return
	}

	// 7. Make a calculation with data from the completed tasks
	var result = ""
	tasks, _ := ccredis.Client.SMembers("tasks/measurement/" + action.UUID).Result()
	if len(tasks) > 0 {
		if action.Action == "ping" {
			// make calculation
			result = processPing(action, tasks)
		} else {
			for _, taskUUID := range tasks {
				tp, _ := ccredis.Client.Get("task/" + taskUUID).Result()
				var subtask Action
				err := json.Unmarshal([]byte(tp), &subtask)
				if err != nil {
					log.Println(err.Error())
				} else if subtask.Result != "" {
					// make calculation
					result += subtask.Result + "\n"
				}
			}
		}
	}

	// 8. Write the result to the main task
	resultAction := Action{MngrUUID: *mngruuid, Action: "result", Result: result, UUID: action.UUID}
	resultjson, _ := json.Marshal(resultAction)

	post("http://"+*addr+"/mngr/task/result", string(resultjson))
}

func processPing(action *Action, tasks []string) (result string) {
	var validCount int
	var validSumm time.Duration
	for _, taskUUID := range tasks {
		tp, _ := ccredis.Client.Get("task/" + taskUUID).Result()
		var subtask Action
		err := json.Unmarshal([]byte(tp), &subtask)
		if err != nil {
			log.Println(err.Error())
		} else if subtask.Result != "" {
			t, err := time.ParseDuration(subtask.Result)
			if err != nil {

			} else {
				validCount++
				validSumm += t
			}
		}
	}
	if validCount == 0 {
		result = "All failed"
	} else {
		result = (time.Duration(validSumm.Nanoseconds()/int64(validCount)) * time.Nanosecond).String()
		if validCount != len(tasks) {
			result += ", " + strconv.FormatInt(int64(len(tasks)-validCount), 10) + " failed"
		}
	}
	return result
}
