package bpa

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func New(input *Input) {
	var path, pipe string

	if len(input.Addi) == 0 || input.Addi[0] == "" {
		log.Fatal("target path is empty string in new mode")
	}

	path, _ = filepath.Abs(filepath.Dir(input.Addi[0]))
	pipe = filepath.Base(input.Addi[0])

	ErrExit(os.MkdirAll(filepath.Join(path, "config"), 0755))
	ErrExit(os.MkdirAll(filepath.Join(path, "tmp"), 0755))

	var bts []byte
	bts, _ = ioutil.ReadFile(input.Args[0])
	ErrExit(ioutil.WriteFile(filepath.Join(path, pipe), bts, 0755))

	ErrExit(ioutil.WriteFile(filepath.Join(path, "project.ini"),
		[]byte(_project_p1+_project_p2), 0642))

	ErrExit(ioutil.WriteFile(filepath.Join(path, "config", "global.ini"),
		[]byte(_global), 0642))

	ErrExit(ioutil.WriteFile(filepath.Join(path, "config", "pipeline.ini"),
		[]byte(_pipeline), 0642))

	ErrExit(ioutil.WriteFile(filepath.Join(path, "config", "tasks1.cfg"),
		[]byte(_tks1), 0642))

	ErrExit(ioutil.WriteFile(filepath.Join(path, "config", "tasks2.cfg"),
		[]byte(_tks2), 0642))

	// for web
	ErrExit(ioutil.WriteFile(filepath.Join(path, "tmp", "run.ini"),
		[]byte(_project_p2), 0642))

	ErrExit(ioutil.WriteFile(filepath.Join(path, "tmp", "run.json"),
		[]byte(_run4web), 0642))

	ErrExit(ioutil.WriteFile(filepath.Join(path, "container.sh"),
		[]byte(_container), 0755))

	fmt.Printf("Template pipeline created: \"%s\"\n", input.Addi[0])
}

var _tks1 string = `#! /bin/bash

[rawqc]
.Type = sample
rawdata = 
Pipeline = {{ .Pipeline }}
evalue = 1E-5
.Cmd = ##
	echo "MAIN: $MAIN"
	echo "RUNNER: $RUNNER"
	echo "Pipeline: $Pipeline"
	echo "Project config: log/$RUNNER/project.ini"
	ls -al log/$RUNNER/project.ini
	echo
	echo $sample, $rawdata, $NC
	## do FastQC
	echo "FastQC"
	sleep $(seq 40 60 | shuf -n 1)
	##
	echo "Done"


[trim]
.Type = sample
rawdata = 
adapter3 = {{ .adapter3 }}
adapter5 = {{ .adapter5 }}
read_minlen = {{ .read_minlen }}
Trimmomatic = {{.Trimmomatic}}
seconds = 
evalue = 1E-10
.Cmd = ##
	echo "$sample trimming..."
	echo $sample, $rawdata, $adapter3, $adapter5, $read_minlen
	##
	if [ -z $seconds ]; then
		echo "seconds of $sample wasn't set, using a random one"
		seconds=$(seq 40 60 | shuf -n 1)
	fi
	##
	sleep $seconds
	echo "done"


[SumReads]
.Type = *sample
G1 = {{ .G1 }}
.Cmd = ##
	echo "G1 = $G1"
	echo "Samples:"
	for i in $sample; do
		echo $i
	done
`

var _tks2 = `#! /bin/bash

[callvariants]
.Type = somatic
.Json = true
evalue = 1E-15
vs = 
.Cmd = ##
	#OIFS=$IFS; IFS=" "; set -- $vs; tumor=$1; normal=$2; IFS=$OIFS 
    p=($vs)
    if [ ${#p[*]} -eq 2 ]; then
      echo "detected tumor and normal!"
    else
      echo "invald tumor and normal: $vs"
      exit 1
    fi
	echo "tumor sample: ${p[0]}"
	echo "normal sample: ${p[1]}"
	##
	ls log/$RUNNER/callvariants@$somatic.json
	echo "Start calling variants"
	seconds=$(seq 30 60 | shuf -n 1)
	sleep $seconds
	##
	echo "Done!"


# without binding .Type
[hello]
.Cmd = echo "hello, world"
	$MAIN read log/$RUNNER/project.ini
	$MAIN read log/$RUNNER/project.ini sample::rawdata
	$MAIN read log/$RUNNER/project.ini sample::rawdata t1
	$MAIN s2j log/$RUNNER/project.ini group
`

var _global string = `
read_minlen = 40
adapter3 = AAAAAAAAAA
adapter5 = TTTTTTTTTT
Trimmomatic = java -Djava.io.tmpdir=./tmp -jar /opt/Trimmomatic-0.36/trimmomatic-0.36.jar
`

var _pipeline string = `[P1]
QC = rawqc, trim SumReads
Align = callvariants
`

var _project_p1 string = `Project = PROJ0001
Version = v1
Pipeline = P1
NC = 20
NG = 12
NP = 4
##
`

var _project_p2 string = `read_minlen = 35

[sample::rawdata]
t1 = rawdata/Sample_123_A
	rawdata/Sample_456_A
t2 = rawdata/Sample_XXXXX
n1 = rawdata/Sample_123_B
n2 = rawdata/Sample_YYYYY

[sample::adapter3]
t1 = AAAAATTTTT
n1 = TTTTTAAAA

[sample::seconds]
t1 = 70

[sample::evalue@trim]
t2 = 1E-20

[somatic::vs]
s1 = t1 n1
s2 = t2 n2


[@trim]
adapter5 = CCCCCCCCCC


[group]
t = t1 t2
n = n1  n2, n3,
`

var _run4web string = `{
    "Name": "R-P-C",
    "Pipeline": "P1",
    "Mode": "run",
    "NC": 8,
    "NG": 8,
    "NP": 6,
    "Timeout": 0,
    "Addi": [
        "."
    ]
}
`

var _container string = `#! /bin/bash
# project: https://github.com/d2jvkpn/biwf
set -eu -o pipefail

MAIN="$(dirname $0)/ABC" # customize it
IMG="centos"             # customize it
MAIN=$(readlink -f $MAIN)

args=($(for v in "$@"; do
  if [[ "$v" == *" "* ]]; then echo "\"$v\""; else echo $v; fi
done))

test $# -eq 0 && { echo "no input args"; exit 1; }

proj=$($MAIN read project.ini "" Project)
test -z "$proj" && { echo "Please set \"Project\" in project.ini"; exit 1; }

container=${proj}_$(printf "%x" $(date +"%s"))
MAINDIR=$(dirname $MAIN); uid=$(id -u)

echo "Creating \"$container\"..."
# echo ${args[*]}
if [[ "$(id -n -u)" == root ]]; then
  docker run --rm --name=$container -v=$PWD:/mnt/HostPath -v=$MAINDIR:$MAINDIR \
  -w /mnt/HostPath $IMG sh -l -c "$MAIN run ${args[*]}"
else
  docker run --rm --name=$container -v=$PWD:/mnt/HostPath -v=$MAINDIR:$MAINDIR \
  $IMG sh -c "useradd -u $uid u$uid && su u$uid -l -c \
  'cd /mnt/HostPath && $MAIN run ${args[*]}'"
fi
`
