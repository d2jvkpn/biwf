package main

import (
	"x/src/bpa"
	"x/src/bpb"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const USAGE = `Binx web service, usage:
  $ binx  leaf  <-main mainpath>  [-work ./]  [-ini project.ini] \
    [-ip 127.0.0.1]  [-node 9000]  [-port 9001]  [-timeout 0]
    # -main main program path
    # -work project work path

  $ binx  node  [-node 9000]  [-port 90001]  [-data ./binx_data]
    # -node set node RPC port
    # -port set node HTTP port
    # -data directory to save json files
`

const LISENSE = `
version: 0.4.14
release: 2019-11-17
project: https://github.com/d2jvkpn/biopipe
lisense: GPLv3 (https://www.gnu.org/licenses/gpl-3.0.en.html)
`

func main() {
	var args *bpb.Args
	args = new(bpb.Args)
	flagSet := flag.NewFlagSet("", flag.ExitOnError)

	flagSet.IntVar(&args.Node, "node", 9000, "node RPC service port")

	flagSet.IntVar(&args.Port, "port", 9001, "node HTTP service port")

	flagSet.StringVar(&args.Ini, "ini", "project.ini",
		"config file to set global varibales, objects and task variables")

	flagSet.StringVar(&args.IP, "ip", "127.0.0.1", "set node ip")

	flagSet.StringVar(&args.MainPath, "main", "",
		"set pipeline main program path(biwf) in leaf mode")

	// not important
	flagSet.StringVar(&args.DataPath, "data", "./binx_data",
		"set directory to save json")

	flagSet.StringVar(&args.WorkPath, "work", "./",
		"set project workpath for runner")

	flagSet.DurationVar(&args.Timeout, "timeout", 0,
		"timeout for leaf to keep connected with node")

	flag.Usage = func() {
		fmt.Println(USAGE)
		flagSet.PrintDefaults()
		fmt.Println(LISENSE)
	}

	if len(os.Args) < 2 || strings.HasPrefix(os.Args[1], "-") {
		flag.Usage()
		os.Exit(2)
	}

	args.Mode = os.Args[1]
	flagSet.Parse(os.Args[2:])

	if flagSet.NArg() > 0 {
		log.Printf("addtional argument(s) found %v\n", flagSet.Args())
		os.Exit(2)
	}

	if args.Mode == "leaf" && args.MainPath == "" {
		log.Printf("-main catn't be empty in leaf mode")
		os.Exit(2)
	}

	args.MainPath, _ = filepath.Abs(args.MainPath)
	args.SelfPath = filepath.Dir(args.MainPath)
	args.WorkPath, _ = filepath.Abs(args.WorkPath)
	args.DataPath, _ = filepath.Abs(args.DataPath)
	args.Ini, _ = filepath.Abs(args.Ini)

	bpa.ErrExit(os.Chdir(args.WorkPath))

	switch args.Mode {
	case "node":
		bpb.Node(args)
	case "leaf":
		bpb.Leaf(args)
	default:
		log.Fatalf("invalid mode %s\n", args.Mode)
	}
}
