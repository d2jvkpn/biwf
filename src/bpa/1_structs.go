package bpa

import (
	"os"
	"sync"
	"time"
)

type Task struct {
	Name, Type    string
	Json          bool `json:"Json,omitempty"`
	Default, Vars map[string]string
	Cmd           string `json:"Cmd,omitempty"`
}

type TkGp struct {
	Name  string
	Steps []string
}

type Object struct {
	Name, Type string
	Attr       map[string]string
}

type Unit struct {
	// Project, Version   string
	MAIN, WorkPath     string
	NC, NG             int
	Name, Type, Object string
	Vars               map[string]string
	Cmd                string
}

type Block struct {
	Type    string
	Tasks   []*Task
	Objects []*Object
	Index   []int // task index of each object
}

type Runner struct {
	Name             string
	Project, Version string
	Pipeline, User   string
	WorkPath         string
	Created          time.Time
	PID              int
	Args             []string
	Resource
	Cancelled   chan string `json:"-"`
	Once        *sync.Once  `json:"-"`
	*os.File    `json:"-"`
	*sync.Mutex `json:"-"`
	End         `json:"-"`
}

type Resource struct {
	NC, NG, NP int
	Timeout    time.Duration
}

type End struct {
	Status  string
	Code    int
	Elapsed string
	EndAt   time.Time
}

// notstart     waiting    -2
// running      -          -1
// done                    0
// failed       falling    1
// interrupted  cancelled  2
// timeout      cancelled  2
// error        -          3

type Input struct {
	Mode, Name, Object string
	Timestamp          int64
	NC, NG, NP         int
	Timeout            time.Duration
	Ini                string
	SelfPath, WorkPath string
	Args, Addi         []string // os.Args[1:], non-flag arguments
}

type DRA struct {
	At              int64
	*sync.WaitGroup `json:"-"`
	*sync.Mutex     `json:"-"`
	Divider         map[string]int `json:"-"`
	Ch              chan struct{}  `json:"-"`
	Counter         map[string]int
	Recorder        map[string]time.Time // task, object, status
}

/* dra.Divider
map[string]int { "NC": 0, "NG": 0, "NP": 0, "NW": 0, "K": 0}
*/

/* dra.Counter
map[string]int {
	"blocks": 0, "tasks": 0, "objects": 0,
	"jobs": 0, "waiting":0, "running": 0, "done": 0,
	"failed": 0, "cancelled": 0, "error": 0,
}
*/
