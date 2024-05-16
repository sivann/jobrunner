#!/bin/bash

mkdir -p tmp
echo "---" >> tmp/cmd.log
date >> cmd.log
echo "args:" $@ >> tmp/cmd.log
echo "env:" >> tmp/cmd.log
env | sort | grep JOB >> tmp/cmd.log
echo "---" >> tmp/cmd.log
echo "here"
cp "${JOBRUNNER_REQUEST_DATA_FN}" tmp/
echo lala 1>&2
/bin/sleep 3
echo "this is the output"
ls -l tmp/jobdata* > $JOBRUNNER_RESPONSE_DATA_FN
