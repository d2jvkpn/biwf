package bpb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/user"
	"regexp"
	"sync"
	"time"
	. "x/src/bpa"
)

func CreateLeaf(args *Args) (lfs *LFS, err error) {
	var (
		client *rpc.Client // temporary client to register
		resp   *http.Response
		bts    []byte
		lfx    *LFX
		u      *user.User
		jreg   *JobReg
		jres   *JobRes
	)

	lfs, lfx = new(LFS), new(LFX)
	lfx.LFS, lfs.LFX = lfs, lfx
	jreg, jres = new(JobReg), new(JobRes)

	log.Printf("Connecting to node: %s, http: %d, rpc: %d\n", args.IP,
		args.Port, args.Node)

	lfs.Node = fmt.Sprintf("ip=%s rpc=%d http=%d", args.IP, args.Node,
		args.Port)

	lfs.PID, lfs.Created, lfs.Main = os.Getpid(), time.Now(), args.MainPath

	lfs.Timeout = args.Timeout

	if u, err = user.Current(); err != nil {
		return
	}

	lfs.User = fmt.Sprintf("uid=%s gid=%s username=%s", u.Uid, u.Gid,
		u.Username)

	if lfs.WorkPath, err = os.Getwd(); err != nil {
		return
	}

	if lfs.Project, err = NewProject(args); err != nil {
		return
	}

	lfs.Name = fmt.Sprintf("serve_%x_%s", lfs.Created.Unix(), lfs.PV("@"))

	if lfs.TaskMap, err = LoadTasks(args.SelfPath + "/config"); err != nil {
		return
	}

	resp, err = http.Get(fmt.Sprintf("http://%s:%d/api/a?item=ip", args.IP, args.Port))
	if err != nil {
		return
	}

	bts, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}

	msg := new(MSG)
	if err = json.Unmarshal(bts, msg); err != nil {
		return
	}

	jreg.IP = msg.List[0]
	log.Printf("Got self IP: %s\n", jreg.IP)

	// create a temp client test node RPC service
	if client, err = rpc.DialHTTP("tcp",
		fmt.Sprintf(args.IP+":%d", args.Node)); err != nil {
		return
	}

	defer func() { client.Close() }()
	// set leaf RPC server

	if lfs.Listener, err = net.Listen("tcp", ":"); err != nil {
		err = fmt.Errorf("failed set lfs.Listener: %s", err)
		return
	}

	jreg.Port = lfs.Listener.Addr().(*net.TCPAddr).Port

	if err = rpc.Register(lfx); err != nil {
		return
	}

	go func() {
		var conn net.Conn
		var err error

		if lfs.Listener == nil { // !! important, avoid infinity loop
			log.Println("RPC listener is closed")
			return
		}

		if conn, err = lfs.Listener.Accept(); err != nil {
			return // continue
		}
		// fmt.Println(conn.RemoteAddr().String())
		go rpc.ServeConn(conn)
	}()

	// call NDX.AddLeaf to register and get uuid and jobx rpc port
	jreg.Project = lfs.Project

	if err = client.Call("NDX.AddLeaf", jreg, jres); err != nil {
		err = fmt.Errorf("node register error: %v", err)
		return
	}
	lfs.UUID = jres.UUID

	lfs.Client, err = rpc.Dial("tcp", fmt.Sprintf(args.IP+":%d", jres.Port))

	if err != nil {
		err = fmt.Errorf("dial jobx RPC error: %v", err)
		return
	}

	// err = lfs.Client.Call("JobX.Echo", &in, &out)
	lfs.Link = fmt.Sprintf("ip=%s rpc=%d client=%d", jreg.IP, jreg.Port,
		jres.Port)

	if err = os.MkdirAll("log/"+lfs.Name, 0755); err != nil {
		return
	}

	lfs.File, err = os.Create(fmt.Sprintf("log/%[1]s/%[1]s.logging", lfs.Name))

	if err != nil {
		return
	}

	if err = JsonTo(lfs, lfs.File, true); err != nil {
		return
	}

	lfs.File.WriteString("\n")

	// ready
	lfs.RBDMap, lfs.Ch = make(map[string]*RBD), make(chan [2]string)
	lfs.WaitGroup = new(sync.WaitGroup)

	lfs.Record("~~~", "leaf registered")

	return
}

func NewProject(args *Args) (pjt *Project, err error) {
	var (
		ok     bool
		pd     map[string]string
		objmap map[string]*Object
		tcfg   map[string]map[string]string
		tmp    string
	)

	pjt = new(Project)
	pjt.WorkPath, pjt.Main = args.WorkPath, args.MainPath

	pjt.Global, err = ReadParam(args.SelfPath+"/config/global.ini", true)
	if err != nil {
		return
	}

	if pd, objmap, tcfg, err = LoadPcfg(args.Ini, true); err != nil {
		return
	}

	pjt.Project = pd["Project"]

	ok, _ = regexp.MatchString("^[A-Z][\\-0-9A-Z]{0,31}$", pjt.Project)
	if !ok {
		err = fmt.Errorf("invalid project name: %s", pjt.Project)
		return
	}

	if pjt.Version = pd["Version"]; pjt.Version == "" {
		pjt.Version = "nil"
	}

	ok, _ = regexp.MatchString("^[a-zA-Z][\\-\\.0-9a-zA-Z]{0,15}", pjt.Version)

	if !ok {
		err = fmt.Errorf("invalid project version: %s", pjt.Version)
		return
	}

	pjt.PipeMap, err = ReadPipeline(args.SelfPath + "/config/pipeline.ini")
	if err != nil {
		return
	}

	pjt.Cfg = Map2Ini(pd, "", nil,
		[]string{"Project", "Version", "Pipeline", "NC", "NG", "NP"})

	if tmp = Tcfg2Ini(tcfg); tmp != "" {
		pjt.Cfg += "\n####\n" + tmp
	}

	if tmp = ObjMap2Ini(objmap); tmp != "" {
		pjt.Cfg += "\n####\n" + tmp
	}

	return
}
