package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"
	"os"
	"time"
	// "io/ioutil"
)

type JobRequest struct {
	Data   string `json:"data"`
	Size   string `json:"size"`
	Id     string `json:"id"`
	UserId string `json:"user_id"`
}

type JobResult struct {
	Data       string `json:"data"`
	Id         string `json:"id"`
	ElapsedSec int    `json:"elapsed_sec"`
	Status     int    `json:"status"`
	Message    int    `json:"message"`
}

type JobPayload struct {
	Request             JobRequest
	Result             JobResult
	JobReady            chan []byte
}

var (
	NumWorkers        = 3 //os.Getenv("NUM_WORKERS")
	HttpListenAddress = "127.0.0.1"
	HttpListenPort    = 8080
)

func DoWork(id int) []byte {
	fmt.Printf("DoWork %v: working\n", id)
	time.Sleep(time.Duration(2000+rand.IntN(5000)) * time.Millisecond)
	rnd := rand.IntN(100)
	fmt.Printf("DoWork %v: work done, value returned: %d\n", id, rnd)

    dat, _ := os.ReadFile("nomis_in.xml")
	return dat
}

func worker(id int, jobs chan JobPayload) {
	for {
		jpl := <-jobs

		fmt.Printf("worker: JobPayload: %+v\n", jpl)
		result := DoWork(id)

		fmt.Println("worker:", id, "returning to JobReady channel, length: ", len(result))
		jpl.JobReady <- result // return result to payload handler
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

        //data seems valid
		jp := JobPayload{Request: jobreq, JobReady: make(chan []byte)}

		fmt.Printf("payloadHandler: sending job %v to jobs channel\n", jobid)
		jobs <- jp 

		fmt.Printf("payloadHandler: Waiting for result on JobReady[%v]\n",jobid)

		jr := <-jp.JobReady

		fmt.Println("payloadHandler: received JobReady from worker, length:", len(string(jr[:])))
		w.WriteHeader(http.StatusOK)
        w.Write(jr[:])
	}
}

func main() {
	jobs := make(chan JobPayload)

	//start worker
	for w := 1; w <= NumWorkers; w++ {
		fmt.Println("main: starting worker ", w)
		go worker(w, jobs)
	}

	go func() {
		http.HandleFunc("/payload/", payloadHandler(jobs))
		err := http.ListenAndServe(HttpListenAddress+":"+strconv.Itoa(HttpListenPort), nil)
		if err != nil {
			log.Println("Starting listening for payload messages on port:", HttpListenPort)
		} else {
			log.Fatalf("An error occured while starting payload server %v", err.Error())
		}
	}()

	for {
        time.Sleep(time.Duration(2000) * time.Millisecond)
	}

}
