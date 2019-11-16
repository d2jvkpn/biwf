package bpa

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func Do(input *Input) {
	var (
		err    error
		code   int
		runner *Runner
		blocks []*Block
	)

	if len(input.Addi) == 0 {
		log.Fatal("no input task in run or test mode")
	}

	runner, err = NewRunner(input.Args, input.Name, input.Timestamp)
	ErrExit(err)

	if blocks, err = DoPrepare(input, runner); err != nil {
		runner.Status, runner.Code = "error", 3
		runner.Close()
		ErrExit(err)
	}

	dra := NewDRA(runner.NC, runner.NG)
	dra.SetCounter(blocks)
	code = runner.Local(blocks, dra, true)
	if code != 0 {
		os.Exit(1)
	}
}

func DoPrepare(input *Input, runner *Runner) (blocks []*Block, err error) {
	var (
		k, v, pstr string
		ok         bool
		pd, params map[string]string
		ts         []string
		objmap     map[string]*Object
		objs       []*Object
		tskmap     map[string]*Task
		tasks      []*Task
		tcfg       map[string]map[string]string
		pipemap    map[string][]*TkGp
		pipe       []*TkGp
	)

	// read project config
	if pd, objmap, tcfg, err = LoadPcfg(input.Ini, true); err != nil {
		return
	}

	runner.Project, runner.Version = pd["Project"], pd["Version"]

	if err = runner.SetResource(input, pd); err != nil {
		return
	}

	// read and update global parameters
	params, err = ReadParam(input.SelfPath+"/config/global.ini", true)
	if err != nil {
		return
	}

	if tskmap, err = LoadTasks(input.SelfPath + "/config"); err != nil {
		return
	}
	// interpolate variables and import .Vars for all tasks
	for k, v = range pd {
		if _, ok = params[k]; !ok {
			params[k] = v // add new variable
		} else if ok && v != "" {
			params[k] = v // update value if not empty
		}
	}

	// select tasks
	ts = input.Addi

	if input.Mode == "run" {
		pipemap, err = ReadPipeline(input.SelfPath + "/config/pipeline.ini")
		if err != nil {
			return
		}

		if pd["Pipeline"] == "" {
			err = fmt.Errorf("\"Pipeline\" isn't set in \"%s\"\n", input.Ini)
			return
		}

		if pipe, ok = pipemap[pd["Pipeline"]]; !ok {
			err = fmt.Errorf("pipeline \"%s\" isn't defined\n", pd["Pipeline"])
			return
		}

		if ts, err = PipelineExtract(pipe, ts, pd["Pipeline"]); err != nil {
			return
		}

		runner.Pipeline = pd["Pipeline"]
	}

	if tasks, err = SelectTasks(tskmap, ts); err != nil {
		return
	}

	if err = UpdateVars(tasks, params, tcfg); err != nil {
		return
	}

	// select objects
	objs, err = SelectObjects(objmap,
		strings.Fields(strings.Replace(input.Object, ",", " ", -1)))

	if err != nil {
		return
	}

	objs = append(objs, ClusterObjects(objs)...)
	objs = append(objs, new(Object))

	// build blocks
	if blocks, err = BuildBlocks(tasks, objs); err != nil {
		return
	}

	if err = runner.Land(); err != nil {
		return
	}

	pstr = Map2Ini(pd, "",
		[]string{"Project", "Version", "Pipeline", "NC", "NG", "NP"}, nil)

	pstr += "\n################################\n" + Tcfg2Ini(tcfg)
	pstr += "\n################################\n" + ObjMap2Ini(objmap)

	ioutil.WriteFile("log/"+runner.Name+"/project.ini", []byte(pstr), 0644)
	return
}
