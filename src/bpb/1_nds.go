package bpb

import (
	"fmt"
	"log"
	"net/http"
	"time"
	. "x/src/bpa"
)

func (nds *NDS) MatchJob(s string) (job *Job, err error) {
	var y interface{}
	var ok bool

	if y, ok = nds.JobMap.Load(s); !ok {
		err = fmt.Errorf("item not found: " + s)
		return
	}

	job, _ = y.(*Job)

	return
}

func (nds *NDS) CutLeaf(pv string) (err error) {
	var job *Job

	if job, err = nds.MatchJob(pv); err != nil {
		return
	}

	rec, reply := NewRecord("leave", "node", "leaf"), ""
	job.Record(rec)
	rec.Error = job.Client.Call("LFX.Leave", &rec.By, &reply)
	err = rec.Error

	if job, err = nds.MatchJob(pv); err == nil || rec.Error != nil {
		rec.Operate, rec.Error = "cutleaf", nil
		nds.NDX.Leave(rec, &reply)
	}

	return
}

func (nds *NDS) Stop() {
	nds.Active = false

	nds.JobMap.Range(func(key, value interface{}) (ok bool) {
		job, _ := value.(*Job)
		pv := job.Project.PV("@")
		log.Println("CutLeaf " + pv)
		nds.CutLeaf(pv)
		return
	})

	nds.Listener.Close()

	nds.EndAt = time.Now()
	path := fmt.Sprintf("%s/node_serve_%x.json", nds.DataPath, nds.EndAt.Unix())

	JsonToFile(nds, path, true)
	log.Printf("Node RPC service shutdown: %d\n", nds.Node)
	log.Printf("Node HTTP service shutdown: %d\n", nds.Port)
}

func (nds *NDS) ReqQuery(item string, ip string) (code int, data interface{}) {
	msg := new(MSG)
	code, data = http.StatusOK, msg

	switch item {
	case "ip":
		msg.Title = "IP"
		msg.List = []string{ip}

	case "leaf":
		msg.Title, msg.List = "leaf list", make([]string, 0, 20)

		nds.JobMap.Range(func(key, value interface{}) (ok bool) {
			job, _ := value.(*Job)
			jr := fmt.Sprintf("%s, %d, %s, records=%d",
				job.Created.Format(time.RFC3339), len(job.SubmitMap),
				job.Project.PV("@"), len(job.Records))

			if StrSliceIndex(msg.List, jr) == -1 {
				msg.List = append(msg.List, jr)
			}

			return
		})

	case "node":
		pvs := make([]string, 0, 20)

		nds.JobMap.Range(func(key, value interface{}) (ok bool) {
			job, _ := value.(*Job)
			pv := job.Project.PV("@")
			if StrSliceIndex(pvs, pv) < 0 {
				pvs = append(pvs, pv)
			}
			return
		})

		msg.Title = "node information"
		msg.List = []string{
			fmt.Sprintf("RPC Port: %d", nds.Node),
			fmt.Sprintf("HTTP Port: %d", nds.Port),
			"DataPath: " + nds.DataPath,
			"Created: " + nds.Created.Format(time.RFC3339),
			fmt.Sprintf("Jobs count: %d", len(pvs)),
		}
	}

	return
}
