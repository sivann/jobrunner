#!/bin/bash

SDIR=$(dirname "$0")
cd $SDIR/..

server="$1" 
file_to_send="$2" 

if [[ $# -ne 2 ]] ; then
    echo "$0 <server ip> <file payload>"
    exit 1
fi
jp=tmp/job_payload.json

e=$(date +%s)
echo -n '{"data":"' > $jp
cat "$file_to_send" | base64 | tr -d '\n'  >> $jp
echo '",' >> $jp
echo "\"id\":\"$e\"}" >> $jp

rm -f tmp/out.json
#curl   -iL --post302 --post301  -X POST  -H "Content-Type: application/json"  "${server}":8080/payload --data '{"data":"a29rbzEyMzQK", "id":"1235"}'
curl   -L --post302 --post301  -X POST  -H "Content-Type: application/json"  "${server}":8080/payload -d @${jp}  -o tmp/out.json

echo "Data:"
cat tmp/out.json | jq -r .data | base64 -d
echo ""
echo "EXIT STATUS:"
cat tmp/out.json | jq .exit_status
echo ""
echo ""
echo "ERROR:"
cat tmp/out.json | jq -r .error | base64 -d
echo ""

echo "OUTPUT:"
cat tmp/out.json | jq -r .output | base64 -d

echo ""
