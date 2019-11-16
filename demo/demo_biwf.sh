#! /bin/bash

set -eu -o pipefail

wd=$PWD

# create a new project named "test/ABC_wf/ABC"
test -d test && rm -r test
# test -f test.zip && rm test.zip
../bin/biwf new test/ABC_wf/ABC

cd test
cp ABC_wf/project.ini ./

## list mode
{
	ABC_wf/ABC list pipeline
	echo
	ABC_wf/ABC list var
	echo
	ABC_wf/ABC list global
	echo
	ABC_wf/ABC list object n1 t1
	echo
	ABC_wf/ABC list cmd trim
} > list_out.txt


## run
ABC_wf/ABC run -np 3 . &

ts=$(date +"%s")
ABC_wf/ABC run -np=4 -name=A -timeout=130s -ts $ts . || true

## recover
runner=$(printf "run_%x_A" $ts)
## previouse timeout is ineffective
ABC_wf/ABC recover $runner

## test
ABC_wf/ABC test -name HELLO hello

wait
cd $wd
