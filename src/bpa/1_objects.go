package bpa

import (
	"fmt"
	"github.com/go-ini/ini"
	"regexp"
	"strings"
)

func LoadPcfg(input string, isPath bool) (kv map[string]string,
	objsmp map[string]*Object, tcfg map[string]map[string]string, err error) {

	var (
		cfg                  *ini.File
		sn, tn, k, v, ot, oa string
		ok                   bool
		i                    int
		tmp                  []string
		options              ini.LoadOptions
	)

	if kv, err = ReadParam(input, isPath); err != nil {
		return
	}

	//
	objs := make([]*Object, 0)
	objsmp = make(map[string]*Object)
	tcfg = make(map[string]map[string]string)

	// options.AllowNestedValues = true
	options.AllowPythonMultilineValues = true

	if isPath {
		cfg, err = ini.LoadSources(options, input)
	} else {
		cfg, err = ini.LoadSources(options, []byte(input))
	}

	if err != nil {
		err = fmt.Errorf("failed to load project config")
		return
	}

	// object config
	// [object_type.attr]
	m1, _ := regexp.Compile("^[_a-zA-Z][_a-zA-Z0-9]*::[_a-zA-Z][_a-zA-Z0-9]*$")

	// [object_type.attr@task]
	m2, _ := regexp.Compile("^[_a-zA-Z][_a-zA-Z0-9]*::" +
		"[_a-zA-Z][_a-zA-Z0-9]*@[_\\.\\-a-zA-Z0-9]*$")

	// [@task] config
	m3, _ := regexp.Compile("^@[_\\.\\-a-zA-Z0-9]*$")

	msg1 := "invalid object attr for \"[%s.%s] %s\" in project config"
	msg2 := "invalid value \"%s\" for key \"%s\" in project config"
	var key *ini.Key

	for _, sect := range cfg.Sections() {
		sn = sect.Name()
		if sn == "DEFAULT" {
			continue
		}

		if m1.Match([]byte(sn)) || m2.Match([]byte(sn)) {
			tmp = strings.SplitN(sn, "::", 2)
			ot, oa = tmp[0], tmp[1]
			for _, key = range sect.Keys() {
				k, v = key.Name(), key.Value()

				if strings.Contains(v, "'''") {
					err = fmt.Errorf(msg1, ot, oa, k)
					return
				}

				var obj *Object

				for i = range objs {
					if objs[i].Type == ot && objs[i].Name == k {
						obj = objs[i]
					}
				}

				if obj == nil {
					obj = &Object{k, ot, make(map[string]string)}
					objs = append(objs, obj)
				}

				obj.Attr[oa] = v
			}
		} else if m3.Match([]byte(sn)) {
			tn = strings.Replace(sn, "@", "", 1)
			if _, ok = tcfg[tn]; !ok {
				tcfg[tn] = make(map[string]string)
			}

			for _, key = range sect.Keys() {
				k, v = key.Name(), key.Value()
				if CheckStr(k) < 1 {
					err = fmt.Errorf(msg2, v, k)
					return
				}
				tcfg[tn][k] = v
			}
		} else {
			// an section can't convert to object's attribution
			tn = " " + sn // pesudo-task
			if _, ok = tcfg[tn]; !ok {
				tcfg[tn] = make(map[string]string)
			}

			for _, key = range sect.Keys() {
				k, v = key.Name(), key.Value()
				if CheckStr(k) < 1 {
					err = fmt.Errorf(msg2, v, k)
					return
				}
				tcfg[tn][k] = v
			}
			// err = fmt.Errorf("section \"%s\" is invalid in project config", sn)
			// return
		}
	}

	msg3 := "object \"%s\" is duplicated in type \"%s\" and \"%s\""

	for i = range objs {
		if _, ok = objsmp[objs[i].Name]; ok {
			err = fmt.Errorf(msg3, objs[i].Name, objs[i].Type,
				objsmp[objs[i].Name].Type)
			return
		}

		objsmp[objs[i].Name] = objs[i]
	}

	return
}

func ClusterObjects(objs []*Object) (nobjs []*Object) {
	mp := make(map[string][]int)

	var (
		i, j int
		k    string
		ok   bool
	)

	for i = range objs {
		k = objs[i].Type

		if _, ok = mp[k]; !ok {
			mp[k] = make([]int, 0)
		}

		mp[k] = append(mp[k], i)
	}

	nobjs = make([]*Object, len(mp))

	i = 0
	for k, _ = range mp {
		strs := make([]string, 0, len(mp[k]))

		for _, j = range mp[k] {
			strs = append(strs, objs[j].Name)
		}

		nobjs[i] = &Object{strings.Join(strs, " "), "*" + k, nil}

		i++
	}

	return
}

func SelectObjects(objsmp map[string]*Object, names []string) (objs []*Object,
	err error) {

	objs = make([]*Object, 0, len(objsmp))
	var (
		k  string
		i  int
		ok bool
	)

	if len(names) == 0 {
		for k, _ = range objsmp {
			objs = append(objs, objsmp[k])
		}
		return
	}

	for i = range names {
		if strings.HasPrefix(names[i], "*") {
			continue
		}

		if _, ok = objsmp[names[i]]; !ok {
			err = fmt.Errorf("object \"%s\" not found", names[i])
			return
		}

		objs = append(objs, objsmp[names[i]])
	}

	return
}

func ObjMap2Ini(objmap map[string]*Object) (out string) {
	var (
		i    int
		k, s string
		ok   bool
		obj  *Object
		ks   []string
	)

	mp := make(map[string]map[string]string)

	for k = range objmap {
		if k == "" || strings.HasPrefix(objmap[k].Type, "*") {
			continue
		}

		obj = objmap[k]
		for s = range obj.Attr {
			// fmt.Println(obj.Type+"::"+s)
			if _, ok = mp[obj.Type+"::"+s]; !ok {
				mp[obj.Type+"::"+s] = make(map[string]string)
			}
			mp[obj.Type+"::"+s][k] = obj.Attr[s]
		}
	}

	ks = make([]string, 0, len(mp))
	for k = range mp {
		ks = append(ks, k)
	}
	SortStrSlice(ks)

	for i, k = range ks {
		out += Map2Ini(mp[k], k, nil, nil)
		if i != len(ks)-1 {
			out += "\n"
		}
	}

	return
}
