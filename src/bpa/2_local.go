package bpa

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func (runner *Runner) Local(blocks []*Block, dra *DRA, acceptSignal bool) int {
	var (
		err                error
		names              []string
		i, j, nc, ng, code int
		status, tmp        string
		block              *Block
		timeout            <-chan time.Time
	)

	shutdown := make(chan struct{})
	signalChannel := make(chan os.Signal)

	if acceptSignal {
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	}

	if runner.Timeout > 0 {
		timeout = time.After(runner.Timeout)
	}

	runner.WriteStatus("running", -1)
	log.Printf("%s, start\n", runner.Name)

	for i = range blocks {
		block = blocks[i]
		names = block.GetTaskNames()

		for j = range names {
			names[j] = fmt.Sprintf("%q", names[j])
		}
		runner.File.WriteString(
			fmt.Sprintf("==> BLOCK %d, %s\n", i+1, block.Type))

		runner.File.WriteString(
			fmt.Sprintf("    Tasks: [%s]\n", strings.Join(names, ", ")),
		)
		names = make([]string, 0, len(block.Objects))
		for j = range block.Objects {
			names = append(names, block.Objects[j].Name)
		}
		for j = range names {
			names[j] = fmt.Sprintf("%q", names[j])
		}
		runner.File.WriteString(
			fmt.Sprintf("    Objects: [%s]\n", strings.Join(names, ", ")),
		)

		names = make([]string, 0, len(block.Index))
		for j = range block.Index {
			names = append(names, fmt.Sprintf("%d", block.Index[j]))
		}

		runner.File.WriteString(
			fmt.Sprintf("    Index: [%s]\n", strings.Join(names, ", ")),
		)
	}

	runner.Record("\n~~~",
		fmt.Sprintf("blocks: %d, tasks: %d, objects: %d, jobs: %d",
			dra.Counter["blocks"], dra.Counter["tasks"],
			dra.Counter["objects"], dra.Counter["jobs"]))

	go func() {
		select {
		case <-timeout:
			runner.WriteStatus("timeout", 2)
			runner.Record("!!!", "runner status: timeout")
			runner.Cancel()
		case <-signalChannel:
			runner.WriteStatus("interrupted", 2)
			runner.Record("!!!", "runner status: interrupted")
			runner.Cancel()
		case <-shutdown:
			return
		}
	}()

	for i = range blocks {
		block = blocks[i]
		names = block.GetTaskNames()

		runner.File.WriteString(fmt.Sprintf("\n~~~ BLOCK %d\n", i+1))
		if len(block.Objects) == 0 {
			continue
		}

		if err = dra.Prepare(runner.NP, len(block.Objects)); err != nil {
			tmp = "failed to allocate resource for block %v: %v"
			runner.Record("!!!", "error", fmt.Sprintf(tmp, names, err))
			runner.WriteStatus("error", 3)
			break
		}

		for j = range block.Objects {
			if status, code = runner.ReadStatus(); code > 1 {
				break
			}

			nc, ng = dra.Alloc()
			go runner.Queue(block, dra, j, nc, ng)
		}

		dra.Wait()

		if runner.Code > 0 {
			status, code = runner.ReadStatus()
			tmp = "skip the next block, runner status: %s %d"
			runner.Record("!!!", fmt.Sprintf(tmp, status, code))
			perc, _ := dra.Progress(32)
			runner.Record("~~~", perc)
			break
		}
	}

	if runner.Code != -1 {
		blocks, _ = TruncateBlocks(blocks)

		tmp = "log/" + runner.Name + "/1_blocks.json"
		JsonToFile(blocks, tmp, false)

		tmp = "log/" + runner.Name + "/1_runner.json"
		JsonToFile(runner, tmp, true)

		tmp = "saved log/%s/{1_blocks.json,1_runner.json}"
		runner.Record("~~~", fmt.Sprintf(tmp, runner.Name))
	}

	close(shutdown)
	runner.Close()
	return runner.Code
}
