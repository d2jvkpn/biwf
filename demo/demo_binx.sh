#! /bin/bash
set -eu -o pipefail

wd=$PWD

if [ $# -gt 0 ]; then
    IP=$1
else
    IP=localhost
fi

rm -r test/log/serve*/ test/log/run_*RPC*/ test/binx_data 2> /dev/null || true
rm test/binx_node.log test/binx_node.log test.zip 2> /dev/null || true

../bin/binx node -data test/binx_data &> test/binx_node.log &

echo "Starting node service..."
sleep 5

../bin/binx leaf -main test/ABC_wf/ABC -work test -ini test/project.ini &> test/binx_leaf.log &

echo "Connecting to node..."
time sleep 5
curl -v -w "\n%{http_code}\n\n" http://$IP:9001/api | jq .
curl http://$IP:9001/api/a?item=node | jq . -
curl http://$IP:9001/api/a?item=leaf | jq . -

curl http://$IP:9001/api/b/PROJ0001/v1/var?item=trim | jq . -
curl -v http://$IP:9001/api/b/PROJ0001/v1/var?item=trim,rawqc | jq .List[] -


curl -X POST http://$IP:9001/api/b/PROJ0001/v1/submit \
  -F "json=@test/ABC_wf/tmp/run.json" \
  -F "cfgfile=@test/ABC_wf/tmp/run.ini" \
  -H "Content-Type: multipart/form-data" | jq . -

sleep 10

jq ".Name=\"B\"" test/ABC_wf/tmp/run.json > test/submit_B.json

curl -X POST http://$IP:9001/api/b/PROJ0001/v1/submit?timeout=5m \
  -F "json=@test/submit_B.json" \
  -F "cfgfile=@test/ABC_wf/tmp/run.ini" \
  -H "Content-Type: multipart/form-data" > tmp.json && jq . tmp.json

R1=$(jq .Title tmp.json | sed 's/"//g'); rm tmp.json
echo "submit echo $R1"

curl http://$IP:9001/api/b/PROJ0001/v1/query?item=record | jq . -

curl http://$IP:9001/api/b/PROJ0001/v1/query?item=submit | jq . -

curl http://$IP:9001/api/b/PROJ0001/v1/query?item=$R1 | jq . -

curl "http://$IP:9001/api/b/PROJ0001/v1/progress?item=$R1" | jq . -

sleep 125

curl -X POST "http://$IP:9001/api/b/PROJ0001/v1/interrupt?item=$R1" | jq . -

curl http://$IP:9001/api/b/PROJ0001/v1/query?item=submit | jq . -

curl "http://$IP:9001/api/b/PROJ0001/v1/progress?item=$R1" | jq . -
##
echo "replaced with $R1"; echo &&
jq ".Name=\"B-recover\" | .Mode=\"recover\" | .Addi = [\"$R1\"] | .NC=12 | .NG=12" \
    test/submit_B.json > test/recover_B.json

curl -X POST http://$IP:9001/api/b/PROJ0001/v1/submit \
  -F "json=@test/recover_B.json" \
  -H "Content-Type: multipart/form-data" > tmp.json

R2=$(jq .Title tmp.json | sed 's/"//g'); rm tmp.json; echo "recovered runner: $R2"

curl "http://$IP:9001/api/b/PROJ0001/v1/progress?item=$R2" | jq . -

curl http://$IP:9001/api/b/PROJ0001/v1/query?item=submit | jq . -

curl http://$IP:9001/api/b/PROJ0001/v1/query?item=record | jq . -

##
time sleep 120
curl -X POST "http://$IP:9001/api/b/PROJ0001/v1/leave" | jq . -

curl http://$IP:9001/api/a?item=leaf | jq . -

test -f test.zip && rm test.zip
zip -r --exclude test/ABC_wf/ABC -q test.zip test

kill %1
