package bpb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	. "x/src/bpa"
)

func (lfs *LFS) NewRecover(subm *Submit) (err error) {
	var bts []byte
	var from, tmp string

	if len(subm.Addi) == 0 {
		err = fmt.Errorf("not runner to recover set")
		return
	}

	from = subm.Addi[0]

	rbd, runner := new(RBD), new(Runner)
	rbd.R = runner

	// blocks
	bts, err = ioutil.ReadFile("log/" + from + "/1_blocks.json")
	if err != nil {
		return
	}

	rbd.B = make([]*Block, 0)

	if err = json.Unmarshal(bts, &rbd.B); err != nil {
		return
	}

	if rbd.B, err = TruncateBlocks(rbd.B); err != nil {
		return
	}

	// runner
	bts, err = ioutil.ReadFile("log/" + from + "/1_runner.json")
	if err != nil {
		return
	}

	if err = json.Unmarshal(bts, rbd.R); err != nil {
		return
	}

	subm.Pipeline = rbd.R.Pipeline
	if err = SetRunner(runner, lfs, subm); err != nil {
		return
	}

	runner.Args = make([]string, 0, 8)

	runner.Args = append(runner.Args, []string{lfs.Main, "recover"}...)
	runner.Args = append(runner.Args, "-nc="+strconv.Itoa(runner.NC))
	runner.Args = append(runner.Args, "-ng="+strconv.Itoa(runner.NG))
	runner.Args = append(runner.Args, "-np="+strconv.Itoa(runner.NP))
	runner.Args = append(runner.Args, "-ts="+strconv.FormatInt(subm.At, 10))

	if subm.Resource.Timeout > 0 {
		tmp = "-timeout=" + subm.Resource.Timeout.String()
		runner.Args = append(runner.Args, tmp)
	}

	runner.Args = append(runner.Args, from)

	// read and update project.ini
	tmp = "log/" + from + "/project.ini"
	if _, err = os.Stat(tmp); err != nil {
		return
	}

	//
	if err = runner.Land(); err != nil {
		return
	}

	if err = FileCopy(tmp, "log/"+runner.Name+"/project.ini"); err != nil {
		return
	}

	subm.Pipeline = ""
	tmp = fmt.Sprintf("log/%s/%s.submit.json", lfs.Name, runner.Name)
	JsonToFile(subm, tmp, true)

	rbd.D = NewDRA(runner.NC, runner.NG)
	rbd.D.SetCounter(rbd.B)

	lfs.RBDMap[runner.Name] = rbd
	lfs.Ch <- [2]string{runner.Name, "recover"}

	return
}
