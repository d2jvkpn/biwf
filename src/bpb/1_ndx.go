package bpb

import (
	"fmt"
	gouuid "github.com/satori/go.uuid"
	"log"
	"net"
	. "x/src/bpa"
	// "net/http"
	"net/rpc"
	"sync"
	"time"
)

// receive leaf register
func (ndx *NDX) AddLeaf(jreg *JobReg, jres *JobRes) (err error) {
	var (
		ok   bool
		uuid string
		job  *Job
		jobx *JobX
	)

	nds := ndx.NDS

	if !nds.Active {
		err = fmt.Errorf("node isn't in active")
		return
	}

	if _, ok = nds.JobMap.Load(jreg.PV("@")); ok {
		err = fmt.Errorf("%s exists", jreg.PV("@"))
		return
	}

	job, jobx = new(Job), new(JobX)
	jobx.Job, job.JobX, job.Created = job, jobx, time.Now()

	// set client
	/*
		job.Client, err = rpc.DialHTTP("tcp",
		fmt.Sprintf("%s:%d", jreg.IP, jreg.Port))
	*/
	job.Client, err = rpc.Dial("tcp", fmt.Sprintf("%s:%d", jreg.IP, jreg.Port))

	if err != nil {
		err = fmt.Errorf("dial leaf rpc error: %v", err)
		return
	}

	// set server
	if job.Listener, err = net.Listen("tcp", ":"); err != nil {
		return
	}

	jres.Port = job.Listener.Addr().(*net.TCPAddr).Port
	// update jobs' fields
	id := gouuid.NewV4()
	uuid = id.String()
	job.UUID, jres.UUID, job.Project = uuid, uuid, jreg.Project
	job.Records = make([]*Record, 0, 20)
	job.SubmitMap, job.Mutex = make(map[string]*Submit), new(sync.Mutex)

	job.Link = fmt.Sprintf("ip=%s rpc=%d client=%d", jreg.IP, jreg.Port,
		jres.Port)

	nds.JobMap.Store(uuid, job)
	nds.JobMap.Store(job.PV("@"), job)

	ser := rpc.NewServer()
	ser.Register(jobx)

	/*
		ser.HandleHTTP(rpc.DefaultRPCPath+"/"+job.UUID,
			rpc.DefaultDebugPath+"/"+job.UUID)

		go http.Serve(job.Listener, nil)
	*/

	go func(pv string) {
		// for {
		var conn net.Conn
		var err error

		if job.Listener == nil {
			// log.Println(pv + " listener closed")
			return
		}

		if conn, err = job.Listener.Accept(); err != nil {
			return //continue
		}

		go ser.ServeConn(conn)
		// }
	}(job.Project.PV("@"))

	job.Record(NewRecord("addleaf", "leaf", "node"))
	log.Printf("AddLeaf %s from %s", job.PV("@"), jreg.IP)

	return
}

func (ndx *NDX) Leave(rec *Record, out *string) (err error) {
	// "removeleaf", by, "project@version"
	var pv, uuid, path string
	var job *Job

	pv = rec.On
	if job, err = ndx.NDS.MatchJob(pv); err != nil {
		return
	}

	uuid = job.UUID

	job.Lock()
	defer job.Unlock()

	job.EndAt = time.Now()
	job.Records = append(job.Records, rec)
	path = fmt.Sprintf("%s/%s@%s.json", ndx.NDS.DataPath, pv, job.UUID)
	JsonToFile(job, path, false)
	log.Printf("%[1]s saved: %[1]s@%[2]s.json", pv, job.UUID)

	if job.Client != nil {
		job.Client.Close()
	}

	if job.Listener != nil {
		job.Listener.Close()
		job.Listener = nil /// assign job.Listener to nil is important
	}

	ndx.NDS.JobMap.Delete(pv)
	ndx.NDS.JobMap.Delete(uuid)

	log.Printf("%s updated: %s called by %s\n", pv, rec.Operate, rec.By)
	// rec.Operate: leave, cutleaf

	return
}
