package bpa

import (
	"fmt"
	"github.com/go-ini/ini"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

func LoadTasks(path string) (tksmp map[string]*Task, err error) {

	tksmp = make(map[string]*Task)

	cfgs, _ := filepath.Glob(path + "/*.cfg")
	if len(cfgs) == 0 {
		err = fmt.Errorf("no .cfg file found in \"%s/config/\"\n", path)
		return
	}

	tksmp, err = ReadTasks(cfgs)

	return
}

func ReadTasks(fs []string) (tksmp map[string]*Task, err error) {
	var (
		cfg     *ini.File
		bts     []byte = make([]byte, 0)
		tmp     []byte
		options ini.LoadOptions
		re      *regexp.Regexp
	)

	// options.AllowNestedValues = true
	options.AllowPythonMultilineValues = true

	tksmp = make(map[string]*Task)

	for i := range fs {
		if tmp, err = ioutil.ReadFile(fs[i]); err != nil {
			return
		}
		bts = append(bts, tmp...)
	}

	if cfg, err = ini.LoadSources(options, bts); err != nil {
		return
	}

	var k, v string

	for _, sect := range cfg.Sections() {
		tk := new(Task)
		tk.Name = sect.Name()
		if tk.Name == "DEFAULT" {
			continue
		}

		if tk.Name == "." || tk.Name == "DEFAULT" {
			err = fmt.Errorf("conflict task name: " + tk.Name)
			return
		}

		if CheckStr(tk.Name) < 0 {
			err = fmt.Errorf("invalid task name \"%s\" in config", tk.Name)
			return
		}

		re, _ = regexp.Compile("(?m)^\\s*\\#+\\s*$")
		tk.Type = sect.Key(".Type").String() // can be empty string
		tk.Cmd = re.ReplaceAllString(sect.Key(".Cmd").String(), "")
		tk.Cmd = strings.TrimSpace(tk.Cmd)
		tk.Default = make(map[string]string)
		tk.Vars = make(map[string]string)

		if sect.Key(".Json").Value() == "true" {
			tk.Json = true
		}

		for _, key := range sect.Keys() {
			k, v = key.Name(), key.Value()
			if k == ".Type" || k == ".Cmd" || k == ".Json" {
				continue
			}

			if CheckStr(k) < 1 {
				err = fmt.Errorf("invalid key name \"%s\" in task \"%s\"",
					k, tk.Name)
				return
			}

			if strings.Contains(v, "'''") {
				msg := "invalid value \"%s\" for key \"%s\" in \"%s\""
				err = fmt.Errorf(msg, v, k, tk.Name)
				return
			}

			tk.Default[k] = v
		}

		tksmp[tk.Name] = tk
	}

	return
}

func SelectTasks(tksmp map[string]*Task, names []string) (tks []*Task, err error) {
	var i int
	var ok bool

	if len(names) == 0 {
		err = fmt.Errorf("no task name for SelectTasks")
		return
	}

	tks = make([]*Task, 0, len(names))

	for i = range names {
		if _, ok = tksmp[names[i]]; !ok {
			err = fmt.Errorf("task \"%s\" is not defined", names[i])
			return
		}

		tks = append(tks, tksmp[names[i]])
	}

	return
}

func UpdateVars(tasks []*Task, gkv map[string]string,
	tcfg map[string]map[string]string) (err error) {
	var i int
	var k, v string
	var re *regexp.Regexp
	var ok bool
	var tk *Task

	re, _ = regexp.Compile("^{{ *\\.[_a-zA-Z][_a-zA-Z0-9]* *}}$")

	for i = range tasks {
		tk = tasks[i]
		for k, v = range tk.Default {
			if !re.Match([]byte(v)) {
				continue
			}

			// value can be empty string as variable not exist in config
			tk.Vars[k] = gkv[strings.Trim(v, "{. }")]
		}
	}

	if len(tcfg) == 0 {
		return
	}

	for i = range tasks {
		tk = tasks[i]

		if _, ok = tcfg[tk.Name]; !ok {
			continue
		}

		for k, v = range tcfg[tk.Name] {
			if _, ok = tk.Default[k]; !ok {
				msg := "variable of task \"%s\" not exists: %s"
				err = fmt.Errorf(msg, tk.Name, k)
				return
			}

			if v != "" {
				tk.Vars[k] = v
			}
		}
	}

	return
}

func Tcfg2Ini(tcfg map[string]map[string]string) (out string) {
	var i int
	var k string
	var ks []string

	ks = make([]string, 0, len(tcfg))

	for k = range tcfg {
		ks = append(ks, k)
	}
	SortStrSlice(ks)

	for i, k = range ks {
		if strings.HasPrefix(k, " ") {
			out += Map2Ini(tcfg[k], strings.TrimLeft(k, " "), nil, nil)
		} else {
			out += Map2Ini(tcfg[k], "@"+k, nil, nil)
		}
		if i != len(ks)-1 {
			out += "\n"
		}
	}

	return
}

func ParamDF(path string, ts []string) (df [][]string, err error) {
	var tksmp map[string]*Task
	var tasks []*Task
	var params map[string]string
	var k, v string
	var i int

	if tksmp, err = LoadTasks(path + "/config"); err != nil {
		return
	}

	if params, err = ReadParam(path+"/config/global.ini", true); err != nil {
		return
	}

	if len(ts) == 0 {
		ts = make([]string, 0, len(tksmp))
		for k, _ = range tksmp {
			ts = append(ts, k)
		}
		SortStrSlice(ts)
	}

	if tasks, err = SelectTasks(tksmp, ts); err != nil {
		return
	}

	UpdateVars(tasks, params, nil)
	params = make(map[string]string)

	for i = range tasks {
		for k, v = range tasks[i].Vars {
			params[strings.Trim(tasks[i].Default[k], "{. }")] = v
		}
	}

	df = make([][]string, 0, len(ts)*5)
	df = append(df, []string{"Name", "Type", "Key", "Value"})

	for k, v = range params {
		df = append(df, []string{"", "", k, v})
	}

	for i = range tasks {
		if len(tasks[i].Default) == 0 {
			df = append(df, []string{tasks[i].Name, tasks[i].Type, "", ""})
			continue
		}

		for k, v = range tasks[i].Default {
			df = append(df, []string{tasks[i].Name, tasks[i].Type, k, v})
		}
	}

	return
}

func (tkgp *TkGp) String() (s string) {
	s = fmt.Sprintf("%s: [%s]", tkgp.Name, strings.Join(tkgp.Steps, ", "))
	return s
}
