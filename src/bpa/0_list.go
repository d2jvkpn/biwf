package bpa

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/tabwriter"
)

func List(input *Input) {
	var (
		err error
		k   string
		i   int
		ok  bool
		ks  []string
	)

	if len(input.Addi) == 0 {
		log.Fatal("not additional args found in list mode")
	}

	switch input.Addi[0] {
	case "global":
		var params, kv map[string]string

		params, err = ReadParam(input.SelfPath+"/config/global.ini", true)
		ErrExit(err)

		if len(input.Addi) == 1 {
			for k = range params {
				fmt.Println(k)
			}
			break
		}

		kv = make(map[string]string)
		for _, k = range input.Addi[1:] {
			if kv[k], ok = params[k]; !ok {
				err = fmt.Errorf("key is not defined: " + k)
				return
			}
		}

		JsonTo(kv, os.Stdout, true)

	case "pipeline":
		var pipemap map[string][]*TkGp

		pipemap, err = ReadPipeline(input.SelfPath + "/config/pipeline.ini")
		ErrExit(err)

		if len(input.Addi) == 1 {
			for k = range pipemap {
				fmt.Println(k)
			}
			break
		}

		for _, k = range input.Addi[1:] {
			if _, ok = pipemap[k]; !ok {
				log.Println("pipeline not found: " + k)
				continue
			}

			ks = make([]string, len(pipemap[k]))
			for i = range pipemap[k] {
				ks[i] = pipemap[k][i].String()
			}

			fmt.Printf("%s = {\n    %s\n}\n\n", k,
				strings.Join(ks, ",\n    "))
		}

	case "cmd":
		var tksmp map[string]*Task
		var tasks []*Task

		tksmp, err = LoadTasks(input.SelfPath + "/config")
		ErrExit(err)

		if len(input.Addi) == 1 {
			for k = range tksmp {
				fmt.Println(k)
			}
			break
		}

		ks = input.Addi[1:]
		tasks, err = SelectTasks(tksmp, ks)
		ErrExit(err)

		for i := range tasks {
			k = strings.Replace(tasks[i].Cmd, "\\n", "\n", -1)
			k = strings.Replace(strings.Trim(k, "\n"), "\n", "\n    ", -1)
			fmt.Printf(">>> %s, %s\n    %s\n\n",
				tasks[i].Name, tasks[i].Type, k)
		}

	case "var":
		var tksmp map[string]*Task
		var df [][]string

		ks = make([]string, len(input.Addi[1:]))
		copy(ks, input.Addi[1:])

		if len(ks) == 0 {
			tksmp, err = LoadTasks(input.SelfPath + "/config")
			if err != nil {
				ErrExit(err)
			}
			for k = range tksmp {
				fmt.Println(k)
			}
			break
		}

		df, err = ParamDF(input.SelfPath, ks)
		ErrExit(err)

		for i = range df {
			if df[i][2] != "" && df[i][3] == "" {
				df[i][3] = "???"
			}
		}

		PrintDF(df, os.Stdout)

	case "object":
		var objsmp map[string]*Object
		var objs []*Object
		var r []string

		_, objsmp, _, err = LoadPcfg(input.Ini, true)
		ErrExit(err)

		if len(input.Addi) == 1 {
			for k = range objsmp {
				fmt.Println(k)
			}
			break
		}
		ks = input.Addi[1:]

		objs, err = SelectObjects(objsmp, ks)
		ErrExit(err)

		array := make([][]string, 0, len(objs)*5)
		array = append(array, []string{"Object", "Type", "Attr", "Value"})

		for i = range objs {
			ks = make([]string, 0, len(objs[i].Attr))
			for k = range objs[i].Attr {
				ks = append(ks, k)
			}

			SortStrSlice(ks)

			for _, k = range ks {
				r = []string{objs[i].Name, objs[i].Type, k, objs[i].Attr[k]}
				array = append(array, r)
			}
		}

		PrintDF(array, os.Stdout)

	case "tcfg":
		var tcfg map[string]map[string]string
		_, _, tcfg, err = LoadPcfg(input.Ini, true)
		ErrExit(err)

		if len(input.Addi) == 1 {
			for k = range tcfg {
				fmt.Println(k)
			}
			break
		}

		tmp := make(map[string]map[string]string)
		for _, t := range input.Addi[1:] {
			tmp[t] = tcfg[t]
		}

		JsonTo(tmp, os.Stdout, true)

	default:
		log.Fatalf("invalid item \"%s\" to print\n", input.Addi[0])
	}

}

func PrintDF(array [][]string, out io.Writer) {
	w := tabwriter.NewWriter(out, 4, 0, 4, ' ', tabwriter.StripEscape)
	for _, r := range array {
		fmt.Fprintln(w, strings.Join(r[:], "\t"))
	}
	w.Flush()
}
