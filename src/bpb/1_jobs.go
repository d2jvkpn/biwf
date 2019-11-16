package bpb

import (
	"fmt"
	"log"
	"regexp"
	. "x/src/bpa"
)

func (job *Job) SubmitNew(subm *Submit) (code int, err error) {
	var (
		ok   bool
		pipe []*TkGp
		msg  string
	)

	code = 1
	if subm.Mode != "run" {
		err = fmt.Errorf("invalid mode: " + subm.Mode)
		return
	}

	ok, _ = regexp.MatchString("^[_\\.\\-a-zA-Z0-9]{1,32}$", subm.Name)
	if !ok {
		err = fmt.Errorf("invalid runner name: " + subm.Name)
		return
	}

	if len(subm.Addi) == 0 {
		err = fmt.Errorf("no input tasks")
		return
	}

	if subm.Mode == "run" {
		if pipe, ok = job.Project.PipeMap[subm.Pipeline]; !ok {
			err = fmt.Errorf("undefined pipeline: " + subm.Pipeline)
			return
		}

		if _, err = PipelineExtract(pipe, subm.Addi, subm.Pipeline); err != nil {
			return
		}
	}

	if subm.Cfg == "" {
		subm.Cfg = job.Project.Cfg // default config
	}

	// make sure submit be added to SubmitMap, and status unchanged by jobx.Update
	job.Lock()
	defer job.Unlock()
	code, msg = 2, ""
	if err = job.Client.Call("LFX.Accept", subm, &msg); err != nil {
		return
	}

	//
	code, subm.End.Status, subm.End.Code = 0, "running", -1
	subm.Name = fmt.Sprintf("run_%x_%s", subm.At, subm.Name)
	rec := NewRecord("run", "node", subm.Name)

	log.Printf("%s submit %s: %s\n", job.Project.PV("@"), "new", subm.Name)
	job.Records = append(job.Records, rec)
	job.SubmitMap[subm.Name] = subm

	return
}

func (job *Job) SubmitRecover(subm *Submit) (code int, err error) {
	var (
		ok        bool
		sub1      *Submit
		from, msg string
	)

	code = 1
	if subm.Mode != "recover" {
		err = fmt.Errorf("invalid mode: " + subm.Mode)
		return
	}

	if len(subm.Addi) == 0 {
		err = fmt.Errorf("runner to recover not set")
		return
	}

	from = subm.Addi[0]

	ok, _ = regexp.MatchString("^[_\\.\\-a-zA-Z0-9]{1,32}$", subm.Name)
	if !ok {
		err = fmt.Errorf("invalid runner name: " + subm.Name)
		return
	}

	if sub1, ok = job.SubmitMap[from]; !ok {
		err = fmt.Errorf("runner(to recover) not found: " + from)
		return
	}

	msg = "runner(to recover) status conflict: %s, %d"
	msg = fmt.Sprintf(msg, sub1.Status, sub1.Code)
	if sub1.Code < -1 {
		err = fmt.Errorf(msg)
		return
	}

	// make sure submit be added to SubmitMap, and status unchanged by jobx.Update
	job.Lock()
	defer job.Unlock()
	code, subm.Cfg, msg = 2, "", ""
	if err = job.Client.Call("LFX.Accept", subm, &msg); err != nil {
		return
	}

	//
	code, subm.End.Status, subm.End.Code = 0, "running", -1
	subm.Name = fmt.Sprintf("run_%x_%s", subm.At, subm.Name)
	rec := NewRecord("recover", "node", subm.Name)

	log.Printf("%s submit %s: %s, %s\n", job.Project.PV("@"), "recover",
		subm.Name, "from "+from)

	job.Records = append(job.Records, rec)
	job.SubmitMap[subm.Name] = subm

	return
}

func (job *Job) Operate(rec *Record) (result string, err error) {
	var ok bool
	var subm *Submit

	if rec.Operate != "interrupt" && rec.Operate != "progress" {
		err = fmt.Errorf("invalid operate")
	}

	if subm, ok = job.SubmitMap[rec.On]; !ok {
		err = fmt.Errorf("submit not found")
		return
	}

	if rec.Operate == "interrupt" {
		tmp := "status conflict (%s, %d)"
		if subm.Code > -1 {
			err = fmt.Errorf(tmp, subm.Status, subm.Code)
			return
		}
	}

	//
	job.Lock()
	defer job.Unlock()

	if err = job.Client.Call("LFX.Operate", rec, &result); err != nil {
		return
	}

	if rec.Operate != "progress" {
		job.Records = append(job.Records, rec)
	}

	return
}

func (job *Job) Record(rec *Record) {
	job.Lock()
	job.Records = append(job.Records, rec)
	job.Unlock()
}
