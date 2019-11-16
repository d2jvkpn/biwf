package bpa

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

func NewDRA(nc, ng int) (dra *DRA) {
	dra = new(DRA)
	dra.WaitGroup = new(sync.WaitGroup)
	dra.Mutex = new(sync.Mutex)

	dra.Divider = map[string]int{
		"NC": Min1(nc), "NG": Min1(ng),
		"NP": 0, "NW": 0, "K": 0,
	}

	dra.Counter = map[string]int{
		"blocks": 0, "tasks": 0, "objects": 0,
		"jobs": 0, "waiting": 0, "running": 0, "ending": 0,
		"done": 0, "failed": 0, "cancelled": 0, "error": 0,
	}

	dra.Recorder = make(map[string]time.Time)

	return
}

func (dra *DRA) Reset(nc, ng int) {
	dra.Divider["NC"], dra.Divider["NG"] = Min1(nc), Min1(ng)
	return
}

func (dra *DRA) Prepare(np, nw int) (err error) {
	msg := "invalid NW field value \"%d\" for Dynamic Resource Allocation"

	if nw < 1 {
		err = fmt.Errorf(msg, nw)
		return
	}

	np = Min1(np)
	tmp := []int{dra.Divider["NC"], dra.Divider["NG"], nw}
	sort.Ints(tmp)
	if np > tmp[0] {
		np = tmp[0]
	}

	dra.Divider["NP"], dra.Divider["NW"] = np, nw
	dra.Ch = make(chan struct{}, np)

	return
}

func (dra *DRA) Alloc() (nc, ng int) {
	dra.Ch <- struct{}{}
	dra.Add(1)
	dra.Lock()
	defer dra.Unlock()

	var n int = dra.Divider["NP"] + 1 - len(dra.Ch) // + 1 to neutralize self
	if dra.Divider["NW"] < n {
		n = dra.Divider["NW"]
	}

	if n == 0 {
		return
	}

	nc, ng = dra.Divider["NC"]/n, dra.Divider["NG"]/n

	if dra.Divider["NC"]%n > 0 && nc > 0 {
		nc++
	}

	if dra.Divider["NG"]%n > 0 && ng > 0 {
		ng++
	}

	dra.Divider["NC"] -= Min1(nc)
	dra.Divider["NG"] -= Min1(ng)
	dra.Divider["NW"]--

	return
}

func (dra *DRA) Release(nc, ng int) {
	dra.Lock()
	dra.Divider["NC"] += nc
	dra.Divider["NG"] += ng
	dra.Unlock()
	dra.Done()
	<-dra.Ch
}

func (dra *DRA) Start(tn, jn string, islast bool) (nc, ng int) {
	// both nc, ng may be 0
	dra.Lock()
	defer dra.Unlock()
	dra.Counter["running"]++
	dra.Counter["waiting"]--
	dra.Recorder["running, "+tn+" @ "+jn] = time.Now()

	if dra.Divider["NW"] != 0 || dra.Divider["NP"] == len(dra.Ch) {
		return
	}

	if dra.Divider["K"] == 0 {
		dra.Divider["K"] = len(dra.Ch)
	}

	dra.Divider["K"] -= dra.Counter["ending"]

	if dra.Divider["K"] <= 0 {
		return
	}

	nc = dra.Divider["NC"] / dra.Divider["K"]
	ng = dra.Divider["NG"] / dra.Divider["K"]

	if dra.Divider["NC"]%dra.Divider["K"] > 0 && dra.Divider["NC"] > 0 {
		nc++
	}

	if dra.Divider["NG"]%dra.Divider["K"] > 0 && dra.Divider["NG"] > 0 {
		ng++
	}

	dra.Divider["NC"] -= nc
	dra.Divider["NG"] -= ng
	dra.Divider["K"]--

	if islast {
		dra.Counter["ending"]++
	}

	return
}

func (dra *DRA) Finished(st, tn, jn string, islast bool) {
	dra.Lock()
	defer dra.Unlock()

	t1 := time.Now()
	e := t1.Sub(dra.Recorder["running, "+tn+" @ "+jn])
	delete(dra.Recorder, "running, "+tn+" @ "+jn)
	dra.Counter["running"]--
	dra.Counter[st]++
	if islast {
		dra.Counter["ending"]--
	}

	if st != "done" {
		dra.Recorder[st+", "+tn+" @ "+jn+", elapsed="+e.String()] = t1
	}
}

func (dra *DRA) SetCounter(blocks []*Block) {
	om := make(map[string]int)
	var j, i, n int
	var blk *Block

	dra.Counter["blocks"] = len(blocks)

	for _, blk = range blocks {
		n = 0
		for j = range blk.Objects {
			i = blk.Index[j]
			dra.Counter["jobs"] += len(blk.Tasks[i:])
			if i < len(blk.Tasks) {
				om[blk.Objects[j].Name] = 1
			}

			if n < len(blk.Tasks)-i {
				n = len(blk.Tasks) - i
			}
		}

		dra.Counter["tasks"] += n
	}

	dra.Counter["waiting"] = dra.Counter["jobs"]
	dra.Counter["objects"] = len(om)

	return
}

func (dra *DRA) Progress(w int) (perc string, records []string) {
	if dra.Counter["jobs"] == 0 {
		perc = "NA"
		return
	}

	dra.Lock()
	records = make([]string, 0, len(dra.Recorder))

	for r := range dra.Recorder {
		t := dra.Recorder[r]

		if strings.HasPrefix(r, "running, ") {
			r += ", actived=" + time.Now().Sub(t).String()
		}

		records = append(records, t.Format(time.RFC3339)+", "+r)
	}

	nd, nr := dra.Counter["done"], dra.Counter["running"]
	nj := dra.Counter["jobs"]
	dra.Unlock()

	p := (float64(nd) + float64(nr)/2) / float64(nj)
	k := int(math.Round(float64(w) * p))
	perc = fmt.Sprintf("%.1f%% (%d+%d*0.5)/%d", p*100, nd, nr, nj)

	if w < 3 {
		return
	}

	w -= 2

	if k == w && nd+nr < nj {
		k = w - 1
	}

	if k == 0 && nd+nr > 0 {
		k = 1
	}

	perc = fmt.Sprintf("[%s%s] %s", strings.Repeat("+", k),
		strings.Repeat("-", w-k), perc)

	return
}

func (dra *DRA) JsonTo(out io.Writer, readable bool) (err error) {
	dra.Lock()
	dra.At = time.Now().Unix()
	err = JsonTo(dra, out, readable)
	dra.Unlock()
	return
}
