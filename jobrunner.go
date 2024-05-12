package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
	"bytes"
	// "io/ioutil"
)

type JobRequest struct {
	Data   string `json:"data"`
	Size   string `json:"size"`
	Id     string `json:"id"`
	UserId string `json:"user_id"`
}

type JobResult struct {
	Data       []byte  `json:"data"`
	Wid        int     `json:"worker_id"`
	ElapsedSec float64 `json:"elapsed_sec"`
	ExitStatus     int     `json:"exit_status"`
	Message    []byte  `json:"message"`
}

type StatusResponse struct {
	NJobs int `json:"queued_jobs"`
}
// job payload includes the input request, and a channel to wait for the worker to write the result to
type JobPayload struct {
	Request  JobRequest
	JobReady chan JobResult
}

var (
	NumWorkers        = 3 //os.Getenv("NUM_WORKERS")
    JobQueueCap          = NumWorkers*10
	HttpListenAddress = "127.0.0.1"
	HttpListenPort    = 8080
)

func exitCode(err error) int {
    if e, ok := err.(interface{ExitCode() int}); ok {
        return e.ExitCode()
    }
    return 0
}

func ExecuteCommand(wid int, cmd1 *exec.Cmd) ([]byte, int, []byte) {
    jr_cmd:= os.Getenv("JOBRUNNER_CMD")
    fmt.Println("worker: JOBRUNNE_CMD:", jr_cmd)

    cmd := exec.Command(jr_cmd)
    cmd.Env = os.Environ()
    cmd.Env = append(cmd.Env, "JOBRUNNER_WORKER_ID="+strconv.Itoa(wid))


	fmt.Printf("ExecuteCommand %v: executin: '%v'\n", wid, jr_cmd)

    var outb, errb bytes.Buffer 
    cmd.Stdout = &outb
    cmd.Stderr = &errb
    err := cmd.Run()

    //fmt.Println("out:", outb.String(), "err:", errb.String())
    exitStatus := exitCode(err)

    data := outb.Bytes()
    cmdErr := errb.Bytes()

    fmt.Println("worker: cmd stdout length: ", len(data))
	fmt.Printf("ExecuteCommand %v: work done, data length:%d\n", wid, len(data))

	return data, exitStatus, cmdErr
}

func worker(wid int, jobs chan JobPayload) {
    cmd := exec.Command("ls")
	for {
		jpl := <-jobs
		fmt.Printf("worker: JobPayload: %+v\n", jpl)

		start := time.Now()
		result_data, result_status, result_message := ExecuteCommand(wid, cmd)
		elapsed := time.Since(start)

		jres := JobResult{Data: result_data, ExitStatus: result_status, Message: result_message, ElapsedSec: elapsed.Seconds(), Wid: wid}

		fmt.Println("worker:", wid, "returning to JobReady channel, length: ", len(result_data))
		jpl.JobReady <- jres // return result to payload handler
	}
}

// we wrap the handler in order to pass arguments (jobs)
func payloadHandler(jobs chan JobPayload) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jobreq JobRequest

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&jobreq)

		if err != nil {
			log.Println("payloadHandler: error decoding json")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		//payload is valid even if json keys were missing
		log.Println("payloadHandler: a valid json POST request received:", jobreq)

		// some json validation
		if len(jobreq.Id) == 0 || len(jobreq.Size) == 0 || len(jobreq.Data) == 0 {
			log.Println("Json keys missing")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Json keys missing\n"))
			return
		}
		jobid, err := strconv.Atoi(jobreq.Id)
		if err != nil {
			log.Println("Job ID not numeric")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Job ID not numeric\n"))
			return
		}


        if len(jobs) >= JobQueueCap {
            fmt.Printf("payloadHandler: denying request for job %d, job queue too large\n",jobid) 
            w.WriteHeader(http.StatusServiceUnavailable)
            fmt.Fprintf(w,"job queue too large (%d)\n",len(jobs))
            return
        }

		//data seems valid
		jp := JobPayload{Request: jobreq, JobReady: make(chan JobResult)}

		fmt.Printf("payloadHandler: sending job %v to jobs channel\n", jobid)

		//send to worker
		jobs <- jp

		fmt.Printf("payloadHandler: waiting for result on JobReady[%v]\n", jobid)

		//wait for worker
		jr := <-jp.JobReady

        //send the result to client
		jr_j, err := json.Marshal(jr)

		fmt.Println("payloadHandler: returning to client JobReady received from worker, length:", len(string(jr_j[:])))
        w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jr_j)
        fmt.Fprintf(w,"\n")
	}
}
func statusHandler(jobs chan JobPayload) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
        sr := StatusResponse {NJobs: len(jobs)}
        sr_j, _ := json.Marshal(sr)

        w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
        w.Write(sr_j)
        fmt.Fprintf(w,"\n")
    }
}
	
func main() {
	jobs := make(chan JobPayload, JobQueueCap)

	//start worker
	for w := 1; w <= NumWorkers; w++ {
		fmt.Println("main: starting worker ", w)
		go worker(w, jobs)
	}

	go func() {
		http.HandleFunc("/payload", payloadHandler(jobs))
		http.HandleFunc("/status", statusHandler(jobs))
		err := http.ListenAndServe(HttpListenAddress+":"+strconv.Itoa(HttpListenPort), nil)
		if err != nil {
			log.Println("Starting listening for payload messages on port:", HttpListenPort)
		} else {
			log.Fatalf("An error occured while starting payload server %v", err.Error())
		}
	}()

	for {
		time.Sleep(time.Duration(5000) * time.Millisecond)
        log.Println("main: job queue length: ",len(jobs))
	}

}
