# jobrunner

Start a limited pool of workers, which accept jobs over HTTP, execute a command, and return the result synchronously on the same HTTP request.

## Server Environment:

* JOBRUNNER_NUM_WORKERS: number of workers to start. Default=3
* JOBRUNNER_CMD: command to execute by worker

## Environment available to the executed command:

* JOBRUNNER_CMD: the CMD itself
* JOBRUNNER_WORKER_ID: worker ID (1, 2, 3, etc) that handled the request
* JOBRUNNER_REQUEST_DATA_FN: a temp file containing the data payload from the web request. It will be self-deleted.
* JOBRUNNER_REQUEST_ID: the ID key supplied in the web request.

## HTTP endpoints

* /payload: where to send cmd input
* /status: service status (e.g. jobs waiting)
* /metrics: prometheus metrics

If number of jobs waiting in queue > JOBRUNNER_NUM_WORKERS * 10, the service returns 503 immediately.

## Compilation

go build

## Sending a request

```
curl   -iL --post302 --post301  -X POST  -H "Content-Type: application/json"  localhost:8080/payload \
  --data '{"data":"a29rbzEyMzQK", "id":"1235"}'
```

Where:

* data: base64 encoded input data, available to $JOBRUNNER_CMD as $JOBRUNNER_REQUEST_DATA_FN file
* id: an ID of your choosing, not used by jobrunner itself. Available as JOBRUNNER_REQUEST_ID env var


Response:

```
{
  "data": "aGVyZQotcnctLS0tLS0tICAxIHNpdmFubiAgMTgwMDMzMzU5MiAgOSAxNCBNYXkgMTY6MTIgdG1wL2pvYmRhdGFfd18xX2lkXzEyMzVfMzU4MzI0MjkyMQotcnctLS0tLS0tICAxIHNpdmFubiAgMTgwMDMzMzU5MiAgOSAxNCBNYXkgMTE6NTYgdG1wL2pvYmRhdGFfd18yX2lkXzEyMzVfMTAzOTE5MzI1OAo=",
  "worker_id": 1,
  "elapsed_sec": 3.146105833,
  "exit_status": 0,
  "message": "bGFsYQo="
}
```

* data: base64 encoded result
* worker_id: worker that processed the request
* exit_status: $JOBRUNNER_CMD exit status
* message: error message by worker or $JOBRUNNER_CMD
