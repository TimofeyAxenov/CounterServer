package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

type CompletedTask struct {
	Id     int `json:"id"`
	Result int `json:"result"`
}

type Task struct {
	Id            int           `json:"id"`
	Arg1          int           `json:"arg1"`
	Arg2          int           `json:"arg2"`
	Operation     string        `json:"operation"`
	OperationTime time.Duration `json:"operation_time"`
}

type ReceivedTask struct {
	Task Task `json:"task"`
}

func main() {
	results := make(chan CompletedTask)
	os.Setenv("COMPUTING_POWER", "5")
	maxg, err := strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	if err != nil {
		panic(err)
	}
	for {
		select {
		case complete := <-results:
			sendtask(complete)
		default:
			curramount := runtime.NumGoroutine()
			if curramount < maxg {
				task := gettask()
				if task.Id == 0 {
					continue
				}
				go CountTask(task, results)
			}
		}
	}
}

func sendtask(task CompletedTask) {
	taskjson, err := json.Marshal(task)
	if err != nil {
		return
	}
	respbody := bytes.NewBuffer(taskjson)
	_, err = http.Post("localhost/internal/task", "application/json", respbody)
	if err != nil {
		return
	}
}

func gettask() Task {
	r, err := http.Get("localhost/internal/task")
	if err != nil {
		return Task{Id: 0}
	}
	task := Task{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&task)
	if err != nil {
		return Task{Id: 0}
	}
	return task
}

func CountTask(task Task, res chan CompletedTask) {
	arg1 := task.Arg1
	arg2 := task.Arg2
	oper := task.Operation
	var result int
	switch oper {
	case "+":
		result = arg1 + arg2
	case "-":
		result = arg1 - arg2
	case "*":
		result = arg1 * arg2
	case "/":
		result = arg1 / arg2
	}
	complete := CompletedTask{
		Id:     task.Id,
		Result: result,
	}
	time.Sleep(task.OperationTime)
	res <- complete
}
