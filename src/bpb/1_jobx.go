package bpb

import (
	"fmt"
	"log"
)

func (jobx *JobX) Update(rst *RStatus, msg *string) (err error) {
	*msg = "OK"
	job := jobx.Job

	if _, ok := job.SubmitMap[rst.Name]; !ok {
		err = fmt.Errorf("runner not found")
		return
	}

	log.Printf("%s updated: %s, %s\n", job.PV("@"), rst.Name, rst.Status)

	job.Lock()
	job.SubmitMap[rst.Name].End = rst.End
	rec := NewRecord("update", "leaf", rst.Name)
	job.Records = append(job.Records, rec)
	job.Unlock()

	return
}
