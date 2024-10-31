/* sivann 2024 */

package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "log/slog"
    "net/http"
    "os"
    "os/exec"
    "strconv"
    "sync/atomic"
    "time"
    "strings"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

type JobRequest struct {
    Data   []byte `json:"data"`
    Id     string `json:"id"`
}

type JobResult struct {
    Data       []byte  `json:"data"`
    Wid        int     `json:"worker_id"`
    ElapsedSec float64 `json:"elapsed_sec"`
    ExitStatus int     `json:"exit_status"`
    Output     []byte  `json:"output"`
    Error      []byte  `json:"error"`
}

type StatusResponse struct {
    QueuedJobs int    `json:"queued_jobs"`
    TotalJobs  uint64 `json:"total_jobs"`
}

type metrics struct {
    QueuedJobs  prometheus.Gauge
    TotalJobs   *prometheus.CounterVec
    JobDuration *prometheus.SummaryVec
}

// job payload includes the input request, and a channel to wait for the worker to write the result to
type JobPayload struct {
    Request  JobRequest
    JobReady chan JobResult
}

var (
    NumWorkers        = 3 //os.Getenv("JOBRUNNER_NUM_WORKERS")
    JobQueueCap       = NumWorkers * 10
    HttpListenAddress = "127.0.0.1"
    HttpListenPort    = "8080"
    TotalJobs         = uint64(0)
    JrPassword        = ""
)


func exitCode(err error) int {
    if e, ok := err.(interface{ ExitCode() int }); ok {
        return e.ExitCode()
    }
    return 0
}

func Infof(format string, args ...any) {
    slog.Default().Info(fmt.Sprintf(format, args...))
}

func PrometheusMetrics(reg prometheus.Registerer) *metrics {
    m := &metrics{
        QueuedJobs: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "queued_jobs",
            Help: "Jobs queued in worker queue.",
        }),
        TotalJobs: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "total_jobs",
                Help: "Number of jobs started, per worker.",
            },
            []string{"worker_id"}, //labels
        ),
        JobDuration: prometheus.NewSummaryVec(
            prometheus.SummaryOpts{
                Name: "job_duration",
                Help: "Job duration in float seconds, per worker.",
            },
            []string{"worker_id"}, //labels
        ),
    }
    reg.MustRegister(m.QueuedJobs)
    reg.MustRegister(m.TotalJobs)
    reg.MustRegister(m.JobDuration)
    return m
}

func ExecuteCommand(wid int, request JobRequest) ([]byte, int, []byte, []byte) {
    var outb, errb bytes.Buffer

    wid_s := strconv.Itoa(wid)
    jr_cmd := os.Getenv("JOBRUNNER_CMD")
    slog.Info("worker", "JOBRUNNER_CMD", jr_cmd)

    slog.Info("ExecuteCommand", "workerId", wid, "command", jr_cmd)

    //create input file
    fnprefix := fmt.Sprintf("jobdata_w_%s_id_%s_",wid_s,request.Id)

    fReq, err := os.CreateTemp("", fnprefix+"req_")
    if err != nil {
        slog.Error("ExecuteCommand: failed to create request temp file", "error", err)
        return []byte{}, -1, []byte(err.Error()), []byte{}
    }

    fRes, err := os.CreateTemp("", fnprefix+"res_")
    if err != nil {
        slog.Error("ExecuteCommand: failed to create response temp file", "error", err)
        return []byte{}, -1, []byte(err.Error()), []byte{}
    }
    slog.Info("ExecuteCommand: created temp files", "worker", wid_s, 
        "request.Id", request.Id, 
        "request_fn", fReq.Name(), 
        "response_fn",fRes.Name())

    defer os.Remove(fReq.Name()) // clean up
    defer os.Remove(fRes.Name()) // clean up

    err = os.WriteFile(fReq.Name(), request.Data,  0666)
    if err != nil {
        slog.Error("ExecuteCommand: failed to write request data into temp file", "error", err)
        return []byte{}, -1, []byte(err.Error()), []byte{}
    }

    //prepare cmd
    cmd := exec.Command(jr_cmd)
    cmd.Env = os.Environ()
    cmd.Env = append(cmd.Env, "JOBRUNNER_WORKER_ID="+strconv.Itoa(wid))
    cmd.Env = append(cmd.Env, "JOBRUNNER_REQUEST_DATA_FN="+fReq.Name())
    cmd.Env = append(cmd.Env, "JOBRUNNER_RESPONSE_DATA_FN="+fRes.Name())
    cmd.Env = append(cmd.Env, "JOBRUNNER_REQUEST_ID="+request.Id)
    cmd.Stdout = &outb
    cmd.Stderr = &errb

    err = cmd.Run()
    if err != nil {
        slog.Error("ExecuteCommand: ignoring cmd.Run() error.", "error", err)
        //return []byte{}, -1, []byte(err.Error())
    }

    exitStatus := exitCode(err)

    output_txt := outb.Bytes()
    error_txt := errb.Bytes()
    Infof("ExecuteCommand: stdout:[%s]\nstderr:[%s]\nexitStatus:[%d]", output_txt, error_txt, exitStatus)
    data, err := os.ReadFile(fRes.Name())

    Infof("ExecuteCommand: work done, worker=%v request data_length=%d, result data_length=%d res_file=%s\n", wid, len(request.Data), len(data), fRes.Name())

    return data, exitStatus, output_txt, error_txt
}

func worker(wid int, jobs chan JobPayload, m metrics) {
    for {
        jpl := <-jobs
        slog.Info("worker: received job", "JobPayload.Request.Id", jpl.Request.Id)
        atomic.AddUint64(&TotalJobs, 1)

        start := time.Now()
        result_data, result_status, output_txt, error_txt := ExecuteCommand(wid, jpl.Request)
        elapsed := time.Since(start)
        m.JobDuration.WithLabelValues(strconv.Itoa(wid)).Observe(float64(elapsed.Seconds()))
        m.TotalJobs.WithLabelValues(strconv.Itoa(wid)).Inc()

        jres := JobResult{Data: result_data, ExitStatus: result_status, Output: output_txt, Error: error_txt, ElapsedSec: elapsed.Seconds(), Wid: wid}

        slog.Info("worker returning data to Jobready channel.", "wid", wid, "result_length", len(result_data))
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
            slog.Error("payloadHandler: error decoding json", "error", err)
            w.WriteHeader(http.StatusBadRequest)
            return
        }

        request_jrpass := r.Header.Get("X-JR-PASSWORD")
        if strings.Trim(request_jrpass,"\r\n") != strings.Trim(JrPassword, "\r\n") {
            log.Println("Invalid password supplied in X-JR-PASSWORD header")
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte("Invalid password supplied in X-JR-PASSWORD header\n"))
            return
        }

        //payload is valid even if json keys were missing
        log.Println("payloadHandler: a valid json POST request received")

        // some json validation
        if len(jobreq.Id) == 0 {
            log.Println("Json key 'id' missing or empty")
            w.WriteHeader(http.StatusBadRequest)
            w.Write([]byte("Json key 'id' missing or empty\n"))
            return
        }


        if len(jobreq.Data) == 0 {
            log.Println("Json key 'data'  missing or empty")
            w.WriteHeader(http.StatusBadRequest)
            w.Write([]byte("Json key 'data' missing or empty\n"))
            return
        }

        /*
        jobid, err := strconv.Atoi(jobreq.Id)
        if err != nil {
            slog.Error("payloadHandler: Job ID not numeric", "error", err)
            w.WriteHeader(http.StatusBadRequest)
            w.Write([]byte("Job ID not numeric\n"))
            return
        }
        */
        jobid := jobreq.Id

        if len(jobs) >= JobQueueCap {
            slog.Error("payloadHandler: denying request, job queue too large.", "queue_length", len(jobs), "jobid", jobid)
            w.WriteHeader(http.StatusServiceUnavailable)
            fmt.Fprintf(w, "job queue too large (%d)\n", len(jobs))
            return
        }

        //data seems valid
        jp := JobPayload{Request: jobreq, JobReady: make(chan JobResult)}

        Infof("payloadHandler: sending job [%v] with request.data length:%d to jobs channel, a worker will pick it up\n", jobid, len(jobreq.Data))

        //send to worker
        jobs <- jp

        Infof("payloadHandler: waiting worker result on JobReady[%v]\n", jobid)

        //wait for worker
        jr := <-jp.JobReady

        //send the result to client
        jr_j, err := json.Marshal(jr)

        Infof("payloadHandler: returning to client JobReady received from worker, length: %v", len(string(jr_j[:])))
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write(jr_j)
        fmt.Fprintf(w, "\n")
    }
}
func statusHandler(jobs chan JobPayload) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        sr := StatusResponse{QueuedJobs: len(jobs), TotalJobs: TotalJobs}
        sr_j, _ := json.Marshal(sr)

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write(sr_j)
        fmt.Fprintf(w, "\n")
    }
}

func EnvConf(env_varname string, default_value string) string {
    env_value, isset := os.LookupEnv(env_varname)
    if isset {
        Infof("EnvConf: using  %s=%s env variable.", env_varname, env_value)
        return env_value
    } else {
        Infof("EnvConf: %s not set, using default (%s).", env_varname, default_value)
        return default_value
    }
}


func main() {

    //prometheus registry
    preg := prometheus.NewRegistry()
    //register metrics
    m := PrometheusMetrics(preg)

    //jobs channel to send jobs requests from http handlers to workers
    jobs := make(chan JobPayload, JobQueueCap)


    NumWorkers,_ = strconv.Atoi(EnvConf("JOBRUNNER_NUM_WORKERS", strconv.Itoa(NumWorkers)))
    HttpListenPort = EnvConf("JOBRUNNER_HTTP_LISTEN_PORT", HttpListenPort)
    HttpListenAddress = EnvConf("JOBRUNNER_HTTP_LISTEN_ADDRESS", HttpListenAddress)
    JrPassword = EnvConf("JOBRUNNER_PASSWORD", JrPassword)


    //validate JOBRUNNER_CMD
    jc := os.Getenv("JOBRUNNER_CMD")
    if len(jc) == 0 {
        fmt.Fprintf(os.Stderr, "JOBRUNNER_CMD is empty\n")
        os.Exit(1)
    }
    path, err := exec.LookPath(jc)
    if err != nil {
        log.Fatal("Cannot find command specified in JOBRUNNER_CMD env var:", jc)
    }
    Infof("path for JOBRUNNER_CMD: %s\n", path)

    //start worker
    for w := 1; w <= NumWorkers; w++ {
        slog.Info("main: starting worker", "workerID", w)
        go worker(w, jobs, *m)
    }

    go func() {
        http.HandleFunc("/payload", payloadHandler(jobs))
        http.HandleFunc("/status", statusHandler(jobs))
        http.Handle("/metrics", promhttp.HandlerFor(preg, promhttp.HandlerOpts{Registry: preg}))

        err := http.ListenAndServe(HttpListenAddress+":"+HttpListenPort, nil)
        if err != nil {
            slog.Info("Starting listening for payload messages.", "port", HttpListenPort)
        } else {
            slog.Error("An error occured while starting payload server", "error", err.Error())
        }
    }()

    //loop and log queue length
    for {
        time.Sleep(time.Duration(2000) * time.Millisecond)
        m.QueuedJobs.Set(float64(len(jobs)))
        //slog.Info("main loop", "queue_length", len(jobs))
    }

}
