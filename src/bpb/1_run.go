package bpb

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	. "x/src/bpa"
)

func (lfs *LFS) NewRunner(subm *Submit) (err error) {
	var (
		ok             bool
		cfg, k, v, tmp string
		pd, params     map[string]string
		objmap         map[string]*Object
		pipe           []*TkGp
		tcfg           map[string]map[string]string
		tasks          []*Task
		objs           []*Object
		ts             []string
	)

	rbd, runner := new(RBD), new(Runner)
	rbd.R = runner
	if err = SetRunner(runner, lfs, subm); err != nil {
		return
	}

	runner.Args = make([]string, 0, 6+len(subm.Addi))
	runner.Args = append(runner.Args, []string{lfs.Main, "run"}...)

	if len(subm.Objects) > 0 {
		tmp = strings.Join(subm.Objects, " ")
		runner.Args = append(runner.Args, []string{"-object", tmp}...)
	}

	if subm.Resource.Timeout > 0 {
		tmp = fmt.Sprintf("%s", subm.Resource.Timeout)
		runner.Args = append(runner.Args, "-timeout="+tmp)
	}

	runner.Args = append(runner.Args, "-ts="+strconv.FormatInt(subm.At, 10))

	runner.Args = append(runner.Args, subm.Addi...)

	// set project.ini
	if pd, objmap, tcfg, err = LoadPcfg(subm.Cfg, false); err != nil {
		return
	}

	pd["Project"], pd["Version"] = runner.Project, runner.Version
	pd["Pipeline"], pd["NC"] = runner.Pipeline, strconv.Itoa(runner.NC)
	pd["NG"], pd["NP"] = strconv.Itoa(runner.NG), strconv.Itoa(runner.NP)

	params = make(map[string]string)
	for k = range lfs.Project.Global {
		params[k] = lfs.Project.Global[k]
	}

	for k, v = range pd {
		if _, ok = params[k]; !ok {
			params[k] = v // add new variable
		} else if ok && v != "" {
			params[k] = v // update value if not empty
		}
	}

	cfg = Map2Ini(pd, "",
		[]string{"Project", "Version", "Pipeline", "NC", "NG", "NP"}, nil)

	cfg += "\n################################\n" + Tcfg2Ini(tcfg)
	cfg += "\n################################\n" + ObjMap2Ini(objmap)

	if pd["Pipeline"] == "" {
		err = fmt.Errorf("\"Pipeline\" isn't set")
		return
	}

	if pipe, ok = lfs.Project.PipeMap[pd["Pipeline"]]; !ok {
		err = fmt.Errorf("pipeline \"%s\" isn't defined\n", pd["Pipeline"])
		return
	}

	if ts, err = PipelineExtract(pipe, subm.Addi, pd["Pipeline"]); err != nil {
		return
	}

	if tasks, err = SelectTasks(lfs.TaskMap, ts); err != nil {
		return
	}

	if err = UpdateVars(tasks, params, tcfg); err != nil {
		return
	}

	if objs, err = SelectObjects(objmap, subm.Objects); err != nil {
		return
	}

	objs = append(objs, ClusterObjects(objs)...)
	objs = append(objs, new(Object))

	if rbd.B, err = BuildBlocks(tasks, objs); err != nil {
		return
	}

	// land
	if err = runner.Land(); err != nil {
		return
	}

	k = fmt.Sprintf("log/%s/%s.submit.json", lfs.Name, runner.Name)
	JsonToFile(subm, k, true)

	ioutil.WriteFile("log/"+runner.Name+"/project.ini", []byte(cfg), 0644)

	rbd.D = NewDRA(runner.NC, runner.NG)
	rbd.D.SetCounter(rbd.B)
	lfs.RBDMap[runner.Name] = rbd

	lfs.Ch <- [2]string{runner.Name, "start"}

	return
}
