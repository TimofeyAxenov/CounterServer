package main

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tidwall/buntdb"
	"orc/pkg/paths"
	"orc/pkg/splitter"
	"os"
	"strconv"
	"strings"
)

func main() {

	os.Setenv("TIME_ADDITION_MS", "1000")
	os.Setenv("TIME_SUBTRACTION_MS", "1000")
	os.Setenv("TIME_MULTIPLICATIONS_MS", "2000")
	os.Setenv("TIME_DIVISIONS_MS", "2000")

	e := echo.New()

	expgroup := e.Group("/api/v1")
	expgroup.Use(middleware.Logger())
	expgroup.Use(middleware.Recover())
	//Read Expression
	expgroup.POST("/calculate", func(c echo.Context) error {
		decoder := json.NewDecoder(c.Request().Body)
		newReadExp := splitter.NewExp{}
		newExp := splitter.Exp{}
		err := decoder.Decode(&newReadExp)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		expid := splitter.NewExpID
		splitter.NewExpID++
		exp := newReadExp.Expression
		if strings.Contains(exp, "^") {
			return echo.NewHTTPError(422, "bad expression")
		}
		newExp.Id = splitter.NewExpID
		newExp.Expression = exp
		newExp.Status = "принято"
		NewDBExp := splitter.DBExp{
			Id:     expid,
			Status: "",
			Result: 0,
		}
		newExpJson, err := json.Marshal(NewDBExp)
		db, err := buntdb.Open(paths.Expsdb)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		defer db.Close()
		err = db.Update(func(tx *buntdb.Tx) error {
			_, _, err := tx.Set(strconv.Itoa(expid), string(newExpJson), nil)
			return err
		})
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		err = splitter.SplitExp(newExp)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		uniqid := splitter.ExpID{Id: expid}
		idjson, err := json.Marshal(uniqid)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		return c.String(201, string(idjson))
	})
	// Write all expressions
	expgroup.GET("/expressions/", func(c echo.Context) error {
		db, err := buntdb.Open(paths.Expsdb)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		defer db.Close()
		exps := make([]string, 0)
		err = db.View(func(tx *buntdb.Tx) error {
			err := tx.Ascend("", func(key, value string) bool {
				exps = append(exps, value)
				return true // continue iteration
			})
			return err
		})
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		writtenexps := splitter.AllExps{
			Expressions: exps,
		}
		staskjson, err := json.Marshal(writtenexps)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		return c.String(200, string(staskjson))
	})
	// Write a specific expression
	expgroup.GET("/expressions/:id", func(c echo.Context) error {
		requestedid := c.Param("id")
		_, err := strconv.Atoi(requestedid)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		db, err := buntdb.Open(paths.Expsdb)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		defer db.Close()
		var gotexp string
		err = db.View(func(tx *buntdb.Tx) error {
			val, err := tx.Get(requestedid)
			gotexp = val
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			switch err.Error() {
			case "ErrNotFound":
				return echo.NewHTTPError(404, err.Error())
			default:
				return echo.NewHTTPError(500, err.Error())
			}
		}
		gotexpstruct := splitter.SpecificExp{Expression: gotexp}
		staskjson, err := json.Marshal(gotexpstruct)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		return c.String(200, string(staskjson))
	})
	taskgroup := e.Group("/internal/task")
	taskgroup.Use(middleware.Logger())
	taskgroup.Use(middleware.Recover())
	// Send a task to an agent
	taskgroup.GET("/", func(c echo.Context) error {
		db, err := buntdb.Open(paths.Tasksdb)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		defer db.Close()
		var amount int
		err = db.View(func(tx *buntdb.Tx) error {
			lenght, err := tx.Len()
			amount = lenght
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		if amount == 0 {
			return echo.NewHTTPError(404, "task not found")
		}
		var taskkey string
		err = db.View(func(tx *buntdb.Tx) error {
			err := tx.Ascend("", func(key, value string) bool {
				taskkey = key
				return false
			})
			return err
		})
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		taskid, _ := strconv.Atoi(taskkey)
		var taskstr string
		err = db.View(func(tx *buntdb.Tx) error {
			val, err := tx.Delete(taskkey)
			taskstr = val
			return err
		})
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		var task splitter.Task
		err = json.Unmarshal([]byte(taskstr), &task)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		senttask := splitter.SentTask{
			Id:            task.Id,
			Arg1:          task.Arg1,
			Arg2:          task.Arg2,
			Operation:     task.Operation,
			OperationTime: task.OperationTime,
		}
		stask := splitter.SendingTask{Task: senttask}
		staskjson, err := json.Marshal(stask)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		splitter.TaskIds = append(splitter.TaskIds, taskid)
		return c.String(200, string(staskjson))
	})
	// Receive a task from an agent
	taskgroup.POST("/", func(c echo.Context) error {
		decoder := json.NewDecoder(c.Request().Body)
		res := splitter.CompletedTask{}
		err := decoder.Decode(&res)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		db, err := buntdb.Open(paths.Existingtasksdb)
		if err != nil {
			return echo.NewHTTPError(500, err.Error())
		}
		var gottask string
		err = db.View(func(tx *buntdb.Tx) error {
			task, err := tx.Get(strconv.Itoa(res.Id))
			gottask = task
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			switch err.Error() {
			case "ErrNotFound":
				return echo.NewHTTPError(404, "task doesn't exist")
			default:
				return echo.NewHTTPError(500, err.Error())
			}
		}
		var recieved splitter.Task
		err = json.Unmarshal([]byte(gottask), &recieved)
		db.Close()
		expid := recieved.TaskExpId
		taskres := res.Result
		taskid := res.Id
		splitter.PlaceTask(expid, taskid, taskres)
		return c.NoContent(200)
	})
	e.Logger.Fatal(e.Start("localhost:8080"))
}
