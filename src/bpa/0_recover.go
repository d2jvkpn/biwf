package bpa

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func Recover(input *Input) {
	var (
		err    error
		code   int
		runner *Runner
		blocks []*Block
	)

	if len(input.Addi) == 0 || input.Addi[0] == "" {
		log.Fatal("no runner to recover was set in recover mode")
	}

	runner = new(Runner)

	blocks, err = RecoverPrepare(input, runner)
	ErrExit(err)

	dra := NewDRA(runner.NC, runner.NG)
	dra.SetCounter(blocks)
	code = runner.Local(blocks, dra, true)
	if code != 0 {
		os.Exit(1)
	}
}

func RecoverPrepare(input *Input, runner *Runner) (blocks []*Block, err error) {
	var bts []byte
	var cfg, tmp string

	tmp = "log/" + input.Addi[0] + "/1_blocks.json"
	if bts, err = ioutil.ReadFile(tmp); err != nil {
		return
	}

	blocks = make([]*Block, 0)
	if err = json.Unmarshal(bts, &blocks); err != nil {
		return
	}

	if blocks, err = TruncateBlocks(blocks); err != nil {
		return
	}

	// set runner
	tmp = "log/" + input.Addi[0] + "/1_runner.json"

	if bts, err = ioutil.ReadFile(tmp); err != nil {
		return
	}

	if err = json.Unmarshal(bts, &runner); err != nil {
		return
	}

	x := strings.SplitN(input.Addi[0], "_", 3)

	if len(x) != 3 {
		err = fmt.Errorf("invalid runner name to recover: " + input.Addi[0])
		return
	}

	if input.Name == "" {
		input.Name = strings.TrimSuffix(x[2], "-recover") + "-recover"
	}

	err = runner.Initialize(input.Args, input.Name, input.Timestamp)
	if err != nil {
		return
	}

	if err = runner.SetResource(input, nil); err != nil {
		return
	}

	cfg = "log/" + input.Addi[0] + "/project.ini"

	if _, err = os.Stat(cfg); err != nil {
		return
	}

	if err = runner.Land(); err != nil {
		return
	}

	err = FileCopy(cfg, "log/"+runner.Name+"/project.ini")

	return
}
