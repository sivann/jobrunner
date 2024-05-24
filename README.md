# jobrunner

Start a limited pool of workers, which accept jobs over HTTP, execute a command, and return the result synchronously on the same HTTP request.

## Server Environment:

* JOBRUNNER_NUM_WORKERS: number of workers to start. Default=3
* JOBRUNNER_CMD: command to execute by worker
* JOBRUNNER_PASSWORD: password for the request

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

Type ```make``` to build for your platform, binaries will be created in bin/ directory. For cross-compiling:

* make linux
* make win

## Sending a request

```
JRP=<same as in $JOBRUNNER_PASSWORD>

curl   -iL --post302 --post301  -X POST \
   -H "Content-Type: application/json" -H "X-JR-PASSWORD: ${JRP}" \
   localhost:8080/payload --data '{"data":"a29rbzEyMzQK", "id":"1235"}'
```

**Where**:

* data: base64 encoded input data, available to $JOBRUNNER_CMD as $JOBRUNNER_REQUEST_DATA_FN file
* id: an ID of your choosing, not used by jobrunner itself. Available as JOBRUNNER_REQUEST_ID env var


**Response**:

```
{
  "data": "aGVyZQotcnctLS0tLS0tICAxIHNpdmFubiAgMTgwMDMzMzU5MiAgOSAxNCBNYXkgMTY6MTIgdG1wL2pvYmRhdGFfd18xX2lkXzEyMzVfMzU4MzI0MjkyMQotcnctLS0tLS0tICAxIHNpdmFubiAgMTgwMDMzMzU5MiAgOSAxNCBNYXkgMTE6NTYgdG1wL2pvYmRhdGFfd18yX2lkXzEyMzVfMTAzOTE5MzI1OAo=",
  "worker_id": 1,
  "elapsed_sec": 3.146105833,
  "exit_status": 0,
  "error": "bGFsYQo=",
  "output": "bGFsYQo="
}
```

* data: base64 encoded result
* worker_id: worker that processed the request
* exit_status: $JOBRUNNER_CMD exit status
* message: error message by worker or $JOBRUNNER_CMD

## Example

Start the server, return output of "date" command:
``` 
JOBRUNNER_HTTP_LISTEN_PORT=8181 JOBRUNNER_CMD=/bin/date  bin/jobrunner
```

Query the server:

```
curl   -sL --post302 --post301  -X POST  -H "Content-Type: application/json" --data '{"data":"a29rbzEyMzQK", "id":"1235"}' localhost:8181/payload |jq -r .output|base64 -d

Fri 24 May 2024 13:59:07 EEST
```

