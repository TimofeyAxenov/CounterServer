package splitter

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/buntdb"
	"orc/pkg/paths"
	"orc/pkg/stack"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

var NewExpID = 1

type NewExp struct {
	Expression string `json:"expression"`
}

type ExpID struct {
	Id int `json:"id"`
}

type Exp struct {
	Id         int    `json:"id"`
	Status     string `json:"status"`
	Expression string `json:"expression"`
}

type DBExp struct {
	Id     int    `json:"id"`
	Status string `json:"status"`
	Result int    `json:"result"`
}

type SentTask struct {
	Id            int           `json:"id"`
	Arg1          int           `json:"arg1"`
	Arg2          int           `json:"arg2"`
	Operation     string        `json:"operation"`
	OperationTime time.Duration `json:"operation_time"`
}

type Task struct {
	Id            int           `json:"id"`
	Arg1          int           `json:"arg1"`
	Arg2          int           `json:"arg2"`
	Operation     string        `json:"operation"`
	OperationTime time.Duration `json:"operation_time"`
	TaskExpId     int
}

type SendingTask struct {
	Task SentTask `json:"task"`
}

type AllExps struct {
	Expressions []string `json:"expressions"`
}

var TaskIds []int

type SpecificExp struct {
	Expression string `json:"expression"`
}

type CompletedTask struct {
	Id     int `json:"id"`
	Result int `json:"result"`
}

type RevPol []string

type SplitExpr struct {
	Id   int      `json:"id"`
	Expr []string `json:"expr"`
}

var prioritiesmap = map[string]int{
	"-": 1,
	"+": 1,
	"*": 2,
	"/": 2,
	"(": 0,
}

var NewTaskID = 1

// ConvertToReversePolish перерабатывает выражение в обратную польскую нотацию (2+2*2->[2, 2, 2, *, +]
func ConvertToReversePolish(exp string) RevPol {
	strings.ReplaceAll(exp, " ", "")
	revpol := make([]string, 0)
	operstack := stack.New()
	symbols := []rune(exp)
	var newnum string
	for i := 0; i < len(symbols); i++ {
		curr := string(symbols[i])
		_, err := strconv.Atoi(curr)
		switch {
		case err == nil:
			newnum += curr
		default:
			revpol = append(revpol, newnum)
			newnum = ""
			if operstack.Len() == 0 {
				operstack.Push(curr)
				continue
			}
			if curr == ")" {
				for {
					newsymb := operstack.Pop()
					if newsymb == "(" {
						break
					}
					revpol = append(revpol, newsymb)
				}
			}
			currprior := prioritiesmap[curr]
			lastsymb := operstack.Peek()
			lastprior := prioritiesmap[lastsymb]
			switch currprior <= lastprior {
			case true:
				addedoper := operstack.Pop()
				revpol = append(revpol, addedoper)
				continue
			case false:
				revpol = append(revpol, curr)
				continue
			}
		}
	}
	for {
		if operstack.Len() == 0 {
			break
		}
		added := operstack.Pop()
		revpol = append(revpol, added)
	}
	return revpol
}

// MakeTasks проверяет реверсивную польскую нотацию на наличие возможных задач
func MakeTasks(expid int) {
	requestedid := strconv.Itoa(expid)
	db, err := buntdb.Open(paths.Splitexpsdb)
	if err != nil {
		panic(err)
	}
	var expr string
	err = db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(requestedid)
		expr = val
		if err != nil {
			return err
		}
		return nil
	})
	db.Close()
	var split SplitExpr
	err = json.Unmarshal([]byte(expr), &split)
	exp := split.Expr
	if len(exp) == 0 {
		res, err1 := strconv.Atoi(exp[0])
		if err1 != nil {
			panic(err)
		}
		FinishExp(expid, res)
	}
	for {
		i := 0
		for i = 0; i < len(exp); i++ {
			curr := exp[i]
			if curr == "+" || curr == "-" || curr == "*" || curr == "/" {
				CreateTask(exp, i, expid)
				break
			}
		}
		if i == len(exp) {
			break
		}
	}
}

// CreateTask проверяет, моно ли создать задачу. Если можно, то создаёт задачу и записывает в базу данных
func CreateTask(exp []string, index int, expid int) {
	oper := exp[index]
	arg1 := exp[index-2]
	arg2 := exp[index-1]
	num1, err := strconv.Atoi(arg1)
	if err != nil {
		return
	}
	num2, err := strconv.Atoi(arg2)
	if err != nil {
		return
	}
	t := MakeTime(oper)
	newtask := Task{
		Id:            NewTaskID,
		Arg1:          num1,
		Arg2:          num2,
		Operation:     oper,
		OperationTime: t,
		TaskExpId:     expid,
	}
	newjson, err := json.Marshal(newtask)
	if err != nil {
		panic(err)
	}
	db, err := buntdb.Open(paths.Tasksdb)
	if err != nil {
		panic(err)
	}
	err = db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(strconv.Itoa(NewTaskID), string(newjson), nil)
		return err
	})
	db.Close()
	db1, err := buntdb.Open(paths.Existingtasksdb)
	if err != nil {
		panic(err)
	}
	err = db1.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(strconv.Itoa(NewTaskID), string(newjson), nil)
		return err
	})
	db1.Close()
	exp[index] = "task" + strconv.Itoa(NewTaskID)
	NewTaskID++
	slices.Delete(exp, index-2, index-1)
}

func MakeTime(oper string) time.Duration {
	var t time.Duration
	switch oper {
	case "+":
		amount := os.Getenv("TIME_ADDITION_MS")
		ms, _ := strconv.Atoi(amount)
		t = time.Millisecond * time.Duration(ms)
	case "-":
		amount := os.Getenv("TIME_SUBTRACTION_MS")
		ms, _ := strconv.Atoi(amount)
		t = time.Millisecond * time.Duration(ms)
	case "*":
		amount := os.Getenv("TIME_MULTIPLICATIONS_MS")
		ms, _ := strconv.Atoi(amount)
		t = time.Millisecond * time.Duration(ms)
	case "/":
		amount := os.Getenv("TIME_DIVISIONS_MS")
		ms, _ := strconv.Atoi(amount)
		t = time.Millisecond * time.Duration(ms)
	}
	return t
}

// PlaceTask обрабатывает ответ на задачу и размещает его в ОПН вместо изначального условия
func PlaceTask(expid int, taskid int, result int) {
	db, err := buntdb.Open(paths.Splitexpsdb)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var expr string
	err = db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Delete(strconv.Itoa(expid))
		expr = val
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	var exp SplitExpr
	err = json.Unmarshal([]byte(expr), &exp)
	taskkey := "task" + strconv.Itoa(taskid)
	expression := exp.Expr
	index := slices.Index(expression, taskkey)
	expression[index] = strconv.Itoa(result)
	exp.Expr = expression
	newjson, err := json.Marshal(exp)
	err = db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(strconv.Itoa(expid), string(newjson), nil)
		return err
	})
	MakeTasks(expid)
}

// SplitExp превращат существующее выражение в выражение в ОПН и заносит в отдельную базу данных
func SplitExp(exp Exp) error {
	id := exp.Id
	expression := exp.Expression
	splitexp := ConvertToReversePolish(expression)
	if len(splitexp) < 3 {
		return fmt.Errorf("not an expression")
	}
	db, err := buntdb.Open(paths.Splitexpsdb)
	if err != nil {
		return err
	}
	defer db.Close()
	splitstr := SplitExpr{Id: id, Expr: splitexp}
	strjs, err := json.Marshal(splitstr)
	if err != nil {
		return err
	}
	err = db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(strconv.Itoa(id), string(strjs), nil)
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

// FinishExp вызывается, когда на выражение есть ответ
func FinishExp(expid int, result int) {
	db, err := buntdb.Open(paths.Expsdb)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var expr string
	err = db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(strconv.Itoa(expid))
		expr = val
		if err != nil {
			return err
		}
		return nil
	})
	var exp DBExp
	err = json.Unmarshal([]byte(expr), &exp)
	if err != nil {
		panic(err)
	}
	exp.Status = "решено"
	exp.Result = result
	newjson, err := json.Marshal(exp)
	err = db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(strconv.Itoa(expid), string(newjson), nil)
		return err
	})
	if err != nil {
		panic(err)
	}
	db1, err := buntdb.Open("")
	if err != nil {
		panic(err)
	}
	err = db1.View(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(strconv.Itoa(expid))
		if err != nil {
			return err
		}
		return nil
	})
}
