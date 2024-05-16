#!/bin/bash

mkdir -p tmp
echo "---" >> tmp/cmd.log
date >> tmp/cmd.log
echo "args:" $@ >> tmp/cmd.log
echo "env:" >> tmp/cmd.log
env | sort | grep JOB >> tmp/cmd.log
echo "---" >> tmp/cmd.log
echo "here"
cp "${JOBRUNNER_REQUEST_DATA_FN}" tmp/
echo -n "THIS IS THE ERROR" 1>&2
/bin/sleep 1
echo -n "this is the output"
ls -l tmp/jobdata* > $JOBRUNNER_RESPONSE_DATA_FN
