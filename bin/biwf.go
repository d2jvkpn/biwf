package main

import (
	"x/src/bpa"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const LISENSE = `
version: 1.3.7
release: 2019-11-17
project: https://github.com/d2jvkpn/biwf
lisense: GPLv3 (https://www.gnu.org/licenses/gpl-3.0.en.html)`

func main() {
	var input *bpa.Input
	input = new(bpa.Input)
	flagSet := flag.NewFlagSet("", flag.ExitOnError)
	bn := filepath.Base(os.Args[0])

	flagSet.StringVar(&input.Object, "object", "", "select objects from "+
		"config, e.g. \"obj1 obj2\" or obj1,obj2")

	flagSet.IntVar(&input.NC, "nc", 8, "total number of cores to use, "+
		"using protject[NC] if not set")

	flagSet.IntVar(&input.NG, "ng", 8, "total amount of memory(GB) to use, "+
		"using project[NG] if not set")

	flagSet.IntVar(&input.NP, "np", 1,
		"maximum parallel number, get value from project[NP] if not set")

	flagSet.StringVar(&input.Ini, "ini", "project.ini", "project config file")

	// not important
	flagSet.StringVar(&input.Name, "name", "", "set third part of runner "+
		"name(^[_\\\\.\\\\-a-zA-Z0-9]{1,32}$), or target name")

	flagSet.DurationVar(&input.Timeout, "timeout", 0,
		"set timeout for runner, e.g. 5h4m3s")

	flagSet.Int64Var(&input.Timestamp, "ts", 0, "set timestamp to use "+
		"fixed runner name(along with -name), e.g. 1563466808")

	flagSet.StringVar(&input.WorkPath, "work", "./",
		"set project workpath for runner")

	flag.Usage = func() {
		fmt.Printf(USAGE["help"], bn)
		fmt.Println(LISENSE)
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(2)
	}

	flagSet.Parse(os.Args[2:])

	input.Mode, input.Addi = os.Args[1], flagSet.Args()
	input.Args = make([]string, len(os.Args))
	copy(input.Args, os.Args)
	input.Args[0], _ = filepath.Abs(input.Args[0])
	input.SelfPath = filepath.Dir(input.Args[0])
	input.Ini, _ = filepath.Abs(input.Ini)

	bpa.ErrExit(os.Chdir(input.WorkPath))

	switch input.Mode {
	case "run", "test":
		if len(input.Addi) == 0 {
			fmt.Printf(USAGE[input.Mode]+"\n", bn)
			flagSet.PrintDefaults()
			os.Exit(2)
		}
	case "recover":
		if len(input.Addi) != 1 {
			fmt.Printf(USAGE[input.Mode]+"\n", bn)
			flagSet.PrintDefaults()
			os.Exit(2)
		}
	case "read", "new", "list", "s2j":
		if len(input.Addi) == 0 {
			fmt.Printf(USAGE[input.Mode]+"\n", bn)
			os.Exit(2)
		}
	default:
		flag.Usage()
		os.Exit(2)
	}

	switch input.Mode {
	case "run", "test":
		bpa.Do(input)
	case "recover":
		bpa.Recover(input)
	case "list":
		bpa.List(input)
	case "read":
		bpa.ReadCfg(input)
	case "new":
		bpa.New(input)
	case "s2j":
		bpa.S2J(input)
	}
}

var USAGE = map[string]string{
	"help": `Bioinformatics workflow of "%[1]s", commands:
  run      run tasks of a pipeline
  recover  recover a stopped runner (using previously saved 1_blocks.json)
  test     run single tasks in sequencial (not interpretate any tasks' groupname)
  list     list data of pipeleline in text or json
  read     read a variable in a ini file
  s2j      convert ini sections to json(split value to an list)
  new      create a pipeline program template
  # use "%[1]s  <command>" for more information about a command
`,
	"run": `  $ %[1]s  run  [options]  <tk1 tk2...>
    options list:
    [-object obj1,obj2]  [-ini project.ini]  [-nc 20]
    [-ng 12]             [-np 4]             [-timeout 24h3m5s]
    [-name May]          [-work ./]          [-ts 1563466808]
    # "." for all task in a pipeline
`,
	"test": `  $ %[1]s  test  [options]  <tk1 tk2...>
    [-object obj1,obj2]  [-ini project.ini]  [-nc 20]
    [-ng 12]             [-np 4]             [-timeout 24h3m5s]
    [-name May]          [-work ./]          [-ts 1563466808]
`,
	"recover": `  $ %[1]s  recover  [options]  <stopped_runner>
    [-nc 20]             [-ng 12]            [-np 4]
    [-timeout 23h4m5s]   [-name June]        [-work ./]
    [-ts 1563466808]
`,
	"list": `  $ %[1]s  list  [-ini project.ini]  <target>  <name1 name2...>
  # flag -ini work when target is "object" only, input and output:
    Target              Source               Format
    golbal k1 k2        global.ini           text
    pipeline P1         pipeline.ini         json
    var tk1 tk2         *.cfg, global.ini    aligned table
    cmd tk1 tk2         *.cfg                text
    tcfg tk1            project.ini          json
    object obj1 obj2    project.ini          aligned table
`,
	"read": `  $ %[1]s  read  <f1.ini,f2.ini...>  [section]  [key]
  # if section or key not found, stdout output is empty and stderr output is 1
`,
	"new": `  $ %[1]s  new  <path/to/pipeline>
`,
	"s2j": `  $ %[1]s  s2j  <f1.ini,f2.ini...>  <sect1 sect2...>
  # convert sections to json (split value to list)
`}
