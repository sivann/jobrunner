# jobrunner

Start a limited pool of workers, which accept jobs over HTTP, execute a command, and return the result synchronously on the same HTTP request.

## Server Environment:

* JOBRUNNER_NUM_WORKERS: number of workers to start. Default=3
* JOBRUNNER_CMD: command to execute by worker

## Environment available to the executed command:

* JOBRUNNER_WORKER_ID: worker ID (1, 2, 3, etc)

## HTTP endpoints

* /payload: where to send cmd input
* /status: service status (e.g. jobs waiting)

If number of jobs waiting in queue > JOBRUNNER_NUM_WORKERS * 10, the service returns 503 immediately.
