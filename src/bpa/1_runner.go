package bpa

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func NewRunner(args []string, bn string, ts int64) (runner *Runner, err error) {
	runner = new(Runner)
	err = runner.Initialize(args, bn, ts)
	return
}

func (runner *Runner) Initialize(args []string, bn string,
	ts int64) (err error) {
	var u *user.User
	var ok bool

	runner.Args, runner.Created = args, time.Now()

	if bn == "" {
		bn = RandomStr(6)
	}

	if ok, _ = regexp.MatchString("^[_\\.\\-a-zA-Z0-9]{1,32}$", bn); !ok {
		err = fmt.Errorf("invalid basename name \"%s\"", bn)
		return
	}

	if ts <= 0 {
		ts = runner.Created.Unix()
	}

	runner.Name = fmt.Sprintf("run_%x_%s", ts, bn)
	runner.Status, runner.Code, runner.PID = "waiting", -2, os.Getpid()

	if runner.WorkPath, err = os.Getwd(); err != nil {
		return
	}

	runner.Mutex, runner.Cancelled = new(sync.Mutex), make(chan string)
	runner.Once = new(sync.Once)

	if u, err = user.Current(); err != nil {
		return
	}

	runner.User = fmt.Sprintf("uid=%s gid=%s username=%s", u.Uid, u.Gid,
		u.Username)

	return
}

func (runner *Runner) Land() (err error) {
	var ok bool

	ok, _ = regexp.MatchString("^[A-Z][\\-0-9A-Z]{0,31}$", runner.Project)

	if runner.Project != "" && !ok {
		err = fmt.Errorf("invalid project name: %s", runner.Project)
		return
	}

	ok, _ = regexp.MatchString("^[a-zA-Z][\\-\\.0-9a-zA-Z]{0,15}",
		runner.Version)

	if runner.Version != "" && !ok {
		err = fmt.Errorf("invalid project version: %s", runner.Version)
		return
	}

	runner.Status = "waiting"
	runner.NC, runner.NG = Min1(runner.NC), Min1(runner.NG)
	runner.NP = Min1(runner.NP)

	if err = os.MkdirAll("log/"+runner.Name, 0755); err != nil {
		return
	}

	bn := fmt.Sprintf("log/%[1]s/%[1]s", runner.Name)
	if runner.File, err = os.Create(bn + ".logging"); err != nil {
		return
	}

	bts, _ := json.MarshalIndent(runner, "", "    ")
	runner.Write(bts)
	runner.WriteString("\n\n")

	return
}

func (runner *Runner) Cancel() {
	runner.Once.Do(func() {
		close(runner.Cancelled)
	})
}

func (runner *Runner) Close() {
	var err error

	switch runner.Code {
	case -1:
		runner.Status, runner.Code = "done", 0
	case -2:
		runner.Status = "notstart"
	case 1:
		runner.Status = "failed"
	}

	runner.EndAt = time.Now()
	runner.Elapsed = fmt.Sprintf("%s", runner.EndAt.Sub(runner.Created))

	if runner.File == nil {
		return
	}

	bts, _ := json.MarshalIndent(runner.End, "", "    ")
	runner.WriteString("\n")
	runner.Write(bts)
	runner.WriteString("\n")
	runner.File.Close()

	bn := fmt.Sprintf("log/%[1]s/%[1]s", runner.Name)

	if runner.Status == "done" {
		err = os.Rename(bn+".logging", bn+".log")
	} else {
		err = os.Rename(bn+".logging", bn+"."+runner.Status)
	}

	if err != nil {
		log.Println(err)
	}

	log.Printf("%s, %s\n", runner.Name, runner.Status)

	return
}

func (runner *Runner) Record(prefix string, strs ...string) {
	var msg string
	var err error

	if len(strs) == 0 {
		return
	}

	msg = fmt.Sprintf("%s %s, %s\n", prefix,
		time.Now().Format("2006-01-02 15:04:05 -0700"),
		strings.Join(strs, ", "))

	runner.Mutex.Lock()

	if runner.File != nil {
		_, err = runner.WriteString(msg)
		if err != nil {
			log.Println(msg)
		}
	} else {
		log.Println(msg)
	}

	runner.Mutex.Unlock()
}

func (runner *Runner) WriteStatus(st string, code int) bool {
	runner.Mutex.Lock()
	defer runner.Mutex.Unlock()

	if code < runner.Code {
		return false
	}

	runner.Status, runner.Code = st, code

	return true
}

func (runner *Runner) ReadStatus() (st string, code int) {
	runner.Mutex.Lock()
	st, code = runner.Status, runner.Code
	runner.Mutex.Unlock()
	return
}

func (runner *Runner) SetResource(input *Input,
	pd map[string]string) (err error) {

	// set runner resources
	if pd["NC"] != "" {
		if runner.NC, err = strconv.Atoi(pd["NC"]); err != nil {
			return
		}
	}

	if pd["NG"] != "" {
		if runner.NG, err = strconv.Atoi(pd["NG"]); err != nil {
			return
		}
	}

	if pd["NP"] != "" {
		if runner.NP, err = strconv.Atoi(pd["NP"]); err != nil {
			return
		}
	}

	if input.NC > 0 {
		runner.NC = input.NC
	}

	if input.NG > 0 {
		runner.NG = input.NG
	}

	if input.NP > 0 {
		runner.NP = input.NP
	}

	runner.Timeout = input.Timeout

	return
}
