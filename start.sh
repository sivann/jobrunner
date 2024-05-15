#!/bin/bash

# sample start script

export JOBRUNNER_NUM_WORKERS=2
export JOBRUNNER_CMD=./jobrunner_cmd_sample.sh
export JOBRUNNER_HTTP_LISTEN_ADDRESS=127.0.0.1
export JOBRUNNER_HTTP_LISTEN_ADDRESS=0.0.0.0

./jobrunner
