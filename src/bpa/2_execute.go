package bpa

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"
)

var basHead = `#! /bin/bash
set -eu -o pipefail
MAIN='''%s'''
MAINDIR='''%s'''
WORKPATH='''%s'''
RUNNER=%s
NC=%d
NG=%d
`

func (runner *Runner) Queue(block *Block, dra *DRA, j, nc, ng int) {
	defer func() {
		dra.Release(nc, ng)
	}()

	var (
		st, jn            string
		i, ncp, ngp, code int
		tk                *Task
		islast            bool
		obj               *Object = block.Objects[j]
	)

	for i = block.Index[j]; i < len(block.Tasks); i++ {
		// _, tk := range block.Tasks[block.Index[j]:]
		// avoid starting a new task whend runner.Status is cancelled
		// "failed" status was caused by a task failure of other objects
		if _, code = runner.ReadStatus(); code > 1 {
			return
		}

		tk, islast = block.Tasks[i], i == len(block.Tasks)-1

		if obj.Type == "" {
			jn = ""
		} else {
			jn = obj.Type + ": " + obj.Name
		}

		ncp, ngp = dra.Start(tk.Name, jn, islast)
		nc, ng = nc+ncp, ng+ngp
		// JsonTo(dra, os.Stdout, true)
		st, code = runner.Execute(tk, obj, nc, ng)
		dra.Finished(st, tk.Name, jn, islast)

		switch code {
		case 1:
			runner.WriteStatus("falling", 1)
			runner.Record("!!!", "now runner is in falling status")
			// perc, _ := dra.Progress(32)
			// runner.Record("~~~", perc)
		case 3:
			runner.WriteStatus("error", 3)
			runner.Cancel()
		}

		if code > 0 {
			break
		} else {
			block.Index[j]++
		}
	}
}

func (runner Runner) Execute(tk *Task, obj *Object, nc, ng int) (st string,
	code int) {
	var bn, codes, name, tmp string
	var err error
	var d map[string]string
	var lwt *os.File

	if obj.Type != "" {
		name = fmt.Sprintf("%s: %s", obj.Type, obj.Name)
	}

	d, codes = runner.Script(tk, obj, nc, ng)
	StartAt := time.Now()

	// write script
	if strings.HasPrefix(obj.Type, "*") {
		tmp = "@" + strings.TrimLeft(obj.Type, "*")
	} else {
		tmp = obj.Name
	}

	bn = fmt.Sprintf("log/%s/%s@%s", runner.Name, tk.Name, tmp)

	if lwt, err = os.Create(bn + ".logging.sh"); err != nil {
		tmp = fmt.Sprintf("failed to create file \"%s\"", bn+".logging.sh")
		runner.Record("!!!", "error", tk.Name, obj.Name, tmp)

		st, code = "error", 3
		return
	}

	if _, err = lwt.WriteString(codes); err != nil {
		st, code = "error", 3
		return
	}

	defer lwt.Close()

	if tk.Json {
		unit := new(Unit)
		// unit.Project, unit.Version = runner.Project, runner.Version
		unit.MAIN, unit.WorkPath = runner.Args[0], runner.WorkPath
		unit.NC, unit.NG = nc, ng
		unit.Name, unit.Type, unit.Object = tk.Name, tk.Type, obj.Name
		unit.Vars, unit.Cmd = d, tk.Cmd

		JsonToFile(unit, bn+".json", true)
	}

	// set command to execute
	cmd := exec.Command("/bin/bash", "./"+bn+".logging.sh")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	rec := make(chan string)
	// note: make sure cmd start before receive cancelled

	lwt.WriteString("StartAt: " + time.Now().Format(time.RFC3339) +
		"\n\n")

	cmd.Stdout, cmd.Stderr = lwt, lwt

	if cmd.Start() != nil {
		runner.Record("!!!", "error", tk.Name, obj.Name, "failed to start")

		st, code = "error", 3
		return
	}

	runner.Record("-->", "start", tk.Name+" @ "+name,
		fmt.Sprintf("pid=%d nc=%d ng=%d", cmd.Process.Pid, nc, ng))

	go func() {
		if cmd.Wait() == nil {
			rec <- "done"
		} else {
			rec <- "failed"
		}

		return
	}()

	select {
	case <-runner.Cancelled:
		st = "cancelled"
		if cmd.Process == nil {
			break
		}

		err = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		// cause rec received a "failed", make sure goroutine closed
		// err = cmd.Process.Kill()
		if err != nil {
			tmp = fmt.Sprintf("failed to kill process %d", cmd.Process.Pid)
			runner.Record("!!!", "error", tk.Name, obj.Name, tmp)
			st = "error"
		}
	case st = <-rec:
	}

	lwt.WriteString("\nEndAt: " + time.Now().Format(time.RFC3339) + "\n")

	switch st {
	case "cancelled":
		code = 2
	case "done":
		code = 0
	case "failed":
		code = 1
	case "error":
		code = 3
	}

	if st == "done" {
		err = os.Rename(bn+".logging.sh", bn+".log.sh")
	} else {
		err = os.Rename(bn+".logging.sh", bn+"."+st+".sh")
	}

	runner.Record("<--", st, tk.Name+" @ "+name,
		fmt.Sprintf("elapsed=%s", time.Now().Sub(StartAt)))

	if err != nil {
		st, code = "error", 3
	}

	return
}

func (runner *Runner) Script(tk *Task, obj *Object,
	nc, ng int) (d map[string]string, codes string) {

	var k, kt string
	var ok bool
	var ks []string

	d = make(map[string]string)

	codes = fmt.Sprintf(basHead+strings.Repeat("#", 64)+"\n",
		runner.Args[0], path.Dir(runner.Args[0]),
		runner.WorkPath, runner.Name, nc, ng)

	if tk.Type != "" {
		k = "%s='''%s'''\n\n"
		codes += fmt.Sprintf(k, strings.TrimLeft(tk.Type, "*"), obj.Name)
		d[strings.TrimLeft(tk.Type, "*")] = obj.Name
	}

	for k = range tk.Default {
		d[k] = tk.Default[k]
		ks = append(ks, k)

		if _, ok = tk.Vars[k]; ok {
			d[k] = tk.Vars[k]
			continue
		}

		if _, ok = obj.Attr[k]; ok && obj.Attr[k] != "" {
			d[k] = obj.Attr[k]
			continue
		}

		kt = k + "@" + tk.Name

		if _, ok = obj.Attr[kt]; ok && obj.Attr[kt] != "" {
			d[k] = obj.Attr[kt]
		}
	}

	SortStrSlice(ks)

	for _, k = range ks {
		codes += fmt.Sprintf("%s='''%s'''\n", k, d[k])
	}

	codes += fmt.Sprintf("%[1]s\n%[2]s\n\nexit\n%[1]s\n",
		strings.Repeat("#", 64), tk.Cmd)

	return
}
