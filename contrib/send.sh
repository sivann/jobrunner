#!/bin/bash


SDIR=$(dirname "$0")
cd $SDIR/..

server="$1" 
file_to_send="$2" 
jrpassword="$3" 

if [[ $# -ne 3 ]] ; then
    echo "$0 <server ip> <file payload> <password>"
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
curl   -L --post302 --post301  -X POST  -H "X-JR-PASSWORD: ${jrpassword}" -H "Content-Type: application/json"  "${server}":8080/payload -d @${jp}   -w "%{http_code}" -o tmp/out.json > tmp/http_code
RET=$?

echo ""
http_code=$(cat tmp/http_code)
echo "HTTP_CODE=$http_code"

echo "CURL EXIT STATUS=$RET"

echo "Data:"
cat tmp/out.json | jq -r .data | base64 -d
echo ""
echo "EXIT STATUS:"
cat tmp/out.json | jq .exit_status
echo ""
echo ""
echo "CMD STDERR:"
cat tmp/out.json | jq -r .error | base64 -d
echo ""

echo "CMD STDOUT:"
cat tmp/out.json | jq -r .output | base64 -d

echo ""
