package bpb

import (
	"github.com/gin-gonic/gin"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
	. "x/src/bpa"
)

type NDS struct {
	Node, Port int // RPC, HTTP port
	DataPath   string
	Created    time.Time
	JobMap     *sync.Map `json:"-"` // UUID, project@version as key, job as value
	NDX        *NDX      `json:"-"`

	Listener net.Listener `json:"-"`
	Engine   *gin.Engine
	Active   bool `json:"-"`
	EndAt    time.Time
}

type NDX struct {
	NDS *NDS
}

type Job struct {
	UUID, Link string
	Created    time.Time
	*Project
	SubmitMap map[string]*Submit
	Records   []*Record
	EndAt     time.Time

	Client      *rpc.Client  `json:"-"`
	Listener    net.Listener `json:"-"`
	JobX        *JobX        `json:"-"`
	*sync.Mutex `json:"-"`
}

type JobX struct {
	Job *Job
}

type JobReg struct {
	*Project
	IP   string
	Port int
}

type JobRes struct {
	UUID string
	Port int
}

type Record struct {
	At      int64
	Operate string
	By, On  string
	Error   error `json:"Error,omitempty"`
}

type RStatus struct {
	Name string
	End
}

type Submit struct {
	Name     string
	Pipeline string
	At       int64 `json:"At,omitempty"`
	Mode     string
	Resource
	Objects []string `json:"Objects,omitempty"` // web default []string{}
	Addi    []string
	Cfg     string `json:"Cfg,omitempty"` // project config text
	End     `gob:"-" json:"-"`
}

////
type LFS struct {
	UUID, Name string
	Node, Link string
	PID        int
	Created    time.Time
	User       string
	WorkPath   string
	Main       string
	Timeout    time.Duration

	*Project `json:"-"`
	TaskMap  map[string]*Task `json:"-"`
	RBDMap   map[string]*RBD  `json:"-"`

	Client          *rpc.Client    `json:"-"`
	Listener        net.Listener   `json:"-"`
	Ch              chan [2]string `json:"-"` // name, operate
	*sync.WaitGroup `json:"-"`
	File            *os.File `json:"-"` // logger
	LFX             *LFX     `json:"-"`
}

type LFX struct {
	LFS *LFS
}

type RBD struct {
	R *Runner
	B []*Block
	D *DRA
}

type MSG struct {
	Title string
	List  []string `json:"List,omitempty"`
}

type Project struct {
	Project, Version string
	WorkPath         string
	Main             string
	PipeMap          map[string][]*TkGp
	Global           map[string]string
	Cfg              string // text
}

type Args struct {
	Mode, IP   string
	Port, Node int // node RPC service port
	DataPath   string
	MainPath   string // pipeline main program path in leaf mode
	WorkPath   string
	SelfPath   string
	Ini        string
	Timeout    time.Duration
}

var API_A_Get = []string{
	"GET /api",
	"GET /api/a?item=[ip, leaf, node]",
}

var API_B_Get = []string{
	"GET /api/b/[$project]/[$version]/query?item=[leaf, record, submit]",
	"GET /api/b/[$project]/[$version]/query?item=[$submitName]",
	"GET /api/b/[$project]/[$version]/progress?item=[$submitName]",
	"GET /api/b/[$project]/[$version]/var?item=[$tk1,$tk2...]",
}

var API_B_POST = []string{
	"POST /api/b/[$project]/[$version]/interrupt?item=[$submitName]",
	"POST /api/b/[$project]/[$version]/submit?timeout=[$timeout]",
	"POST /api/b/[$project]/[$version]/leave",
}
