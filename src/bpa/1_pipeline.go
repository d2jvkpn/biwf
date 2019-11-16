package bpa

import (
	"fmt"
	"github.com/go-ini/ini"
	"strings"
)

func ReadPipeline(fn string) (pipemp map[string][]*TkGp, err error) {
	var cfg *ini.File
	var sn string

	var options ini.LoadOptions
	options.AllowPythonMultilineValues = true
	// options.AllowNestedValues = true

	if cfg, err = ini.LoadSources(options, fn); err != nil {
		err = fmt.Errorf("failed load config \"%s\": %s", fn, err.Error())
		return
	}

	pipemp = make(map[string][]*TkGp)

	var msg1 string = "invalid pipeline name \"%s\" in config \"%s\""
	var msg2 string = "task \"%s\" is duplicated in pipeline \"%s\""

	for _, sect := range cfg.Sections() {
		sn = sect.Name()
		if sn == "DEFAULT" {
			continue
		}

		if CheckStr(sn) < 0 {
			err = fmt.Errorf(msg1, sn, fn)
			return
		}

		pipe := make([]*TkGp, 0, 5)
		all := make([]string, 0, 10)

		for _, k := range sect.Keys() {
			tkgp := new(TkGp)
			tkgp.Name = k.Name()
			tkgp.Steps = strings.Fields(strings.Replace(k.Value(), ",", " ", -1))
			if len(tkgp.Steps) == 0 {
				err = fmt.Errorf("[%s][%s] is empty in %s", sn, k.Name(), fn)
				return
			}
			pipe, all = append(pipe, tkgp), append(all, tkgp.Steps...)

			for i := 0; i < len(all)-1; i++ {
				if HasElem(all[i+1:], all[i]) {
					err = fmt.Errorf(msg2, all[i], sn)
					return
				}
			}
		}

		pipemp[sn] = pipe
	}

	return
}

func PipelineExtract(pipe []*TkGp, names []string, pn string) (out []string,
	err error) {
	if len(names) == 0 {
		err = fmt.Errorf("no task input to select")
		return
	}

	var i int
	var n string

	all := make([]string, 0, len(pipe)*3)
	result := make([]string, 0, len(names)*2)

	for i = range pipe {
		all = append(all, pipe[i].Steps...)
	}

	if names[0] == "." {
		out = all
		return
	}

next:
	for _, n = range names {
		for i = range pipe {
			if n == pipe[i].Name {
				result = append(result, pipe[i].Steps...)
				continue next
			}
			if StrSliceIndex(pipe[i].Steps, n) > -1 {
				result = append(result, n)
				continue next
			}
		}

		err = fmt.Errorf("task \"%s\" is not exists in pipeline \"%s\"", n, pn)
		return
	}

	out = make([]string, 0, len(result))

	for i = range all {
		if StrSliceIndex(result, all[i]) > -1 {
			out = append(out, all[i])
		}
	}

	return
}
