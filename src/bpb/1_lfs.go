package bpb

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	. "x/src/bpa"
)

func (lfs *LFS) Record(prefix string, strs ...string) {
	var tmp string
	tmp = prefix + " " + time.Now().Format("2006-01-02 15:04:05 -0700")

	if lfs.File != nil {
		lfs.File.WriteString(tmp + ", " + strings.Join(strs, ", ") + "\n")
	} else {
		log.Println(strings.Join(strs, ", "))
	}
}

func (lfs *LFS) InterruptAll() {
	var rbd *RBD

	for _, rbd = range lfs.RBDMap {
		if rbd.R.Status == "falling" || rbd.R.Code < 0 {
			lfs.Record("<--", "interrupt", "leaf", rbd.R.Name, "nil")
			rbd.R.WriteStatus("interrupted", 2)
			rbd.R.Cancel()
		}
	}

	lfs.WaitGroup.Wait() // waitting all runner to exit
}

func (lfs *LFS) Stop(by string) {
	var (
		err       error
		prefix, d string
		t         time.Time
	)

	lfs.WaitGroup.Wait() // waitting all runner to exit

	t = time.Now()
	d = t.Sub(lfs.Created).String()

	if lfs.File != nil {
		lfs.Record("~~~", "stopped by "+by)

		lfs.File.Write([]byte("\n"))

		JsonTo(&struct {
			Elapsed string
			EndAt   time.Time
		}{d, t}, lfs.File, true)

		lfs.File.Close()

		prefix = fmt.Sprintf("log/%[1]s/%[1]s", lfs.Name)

		if err = os.Rename(prefix+".logging", prefix+".log"); err != nil {
			log.Println(err)
		}
	}

	if lfs.Client != nil {
		lfs.Client.Close()
	}

	if lfs.Listener != nil {
		lfs.Listener.Close()
		lfs.Listener = nil
	}

	return
}

func (lfs *LFS) Run(msg [2]string) (err error) {
	var (
		ok  bool
		tmp string
		rbd *RBD
	)

	if rbd, ok = lfs.RBDMap[msg[0]]; !ok {
		err = fmt.Errorf("runner not found")
		lfs.Record("!!!", "run", msg[0], msg[1], err.Error())
		return
	}

	lfs.WaitGroup.Add(1)
	defer func() { lfs.WaitGroup.Add(-1) }()

	rbd.R.Status, rbd.R.Code = "running", -1
	t := "task: " + strconv.Itoa(rbd.D.Counter["tasks"])
	o := "object: " + strconv.Itoa(rbd.D.Counter["objects"])
	lfs.Record(">>>", "start", rbd.R.Name, t, o)

	rbd.R.Local(rbd.B, rbd.D, false)
	rst := &RStatus{rbd.R.Name, rbd.R.End}

	lfs.Record("<<<", "end", rbd.R.Name, rbd.R.Status,
		strconv.Itoa(rbd.R.Code))

	tmp = ""
	err = lfs.Client.Call("JobX.Update", rst, &tmp)

	tmp = "%s call JobX.Update error: %s"
	if err != nil {
		lfs.Record("!!!", fmt.Sprintf(tmp, rbd.R.Name, err.Error()))
		return
	}

	return
}

func SetRunner(runner *Runner, lfs *LFS, subm *Submit) (err error) {
	var ok bool

	ok, _ = regexp.MatchString("^[_\\.\\-a-zA-Z0-9]{1,32}$", subm.Name)

	if !ok {
		err = fmt.Errorf("invalid run name \"%s\"", subm.Name)
		return
	}

	runner.Name = fmt.Sprintf("run_%x_%s", subm.At, subm.Name)
	runner.Pipeline, runner.Status, runner.Code = subm.Pipeline, "waiting", -2
	runner.Created, runner.Resource = time.Unix(subm.At, 0), subm.Resource

	// set defaults
	runner.Mutex, runner.Cancelled = new(sync.Mutex), make(chan string)
	runner.Once = new(sync.Once)
	runner.Project, runner.Version = lfs.Project.Project, lfs.Project.Version
	runner.User, runner.PID, runner.WorkPath = lfs.User, lfs.PID, lfs.WorkPath

	return
}
