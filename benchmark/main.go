package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/urfave/cli"
)

type BenchmarkConfig struct{
	Host             string
	Port             int
	OlPath           string
	RegistryMachines []string
	Cmds             []Command
}

type Command struct{
	Cmd        	string
	NumThreads 	int
	TotalCmds  	int
	NumLambdas	int
}

type WorkerPool struct{
	workerID   int
	numWorkers int
	numJobs    int
	done       chan bool
	wg         *sync.WaitGroup
	chls       chan PerfMetrics
	jobs       chan Job
}

type Job struct{
	jobID		int
	url			string
	lambda 		string
	cmds 		string
}

type PerfMetrics struct{
	jobID		int
	lambda		string
	runTime 	int64
}


var LAMBDA_BASE = "hello_"
var BENCHMARK_PREPARE_SCRIPT = "/mnt/lambda_scheduler/s19-lambda/benchmark/benchmark_prepare.sh"
var RUN_LAMBDA_BASE = "/runLambda/"

// WorkerPools consists a list of WorkerPool, one WorkerPool for each
// Command in the corresponding config file.
var WorkerPools []*WorkerPool
var Metrics		[]PerfMetrics

func startBenchmark(ctx *cli.Context) error {
	conf := ctx.String("config")
	if conf == "" {
		fmt.Println("Please supply a valid config file")
	}

	benchmarkConfig, err := readConfig(conf)
	if err != nil {
		fmt.Println("Unable to parse config file, benchmark aborted")
		log.Fatalln(err)
		return err
	}

	//cmdLineOut, err := prepBenchmark(*benchmarkConfig)
	//fmt.Printf("%s", cmdLineOut)
	//if err != nil {
	//	fmt.Println("Failed to execute prepare_benchmark.sh")
	//	log.Fatalln(err)
//		return err
//	}

	genWorkload(*benchmarkConfig)

	return nil
}

func readConfig(config string) (*BenchmarkConfig, error) {
	benchmarkConfig := &BenchmarkConfig{}
	configBytes, err := ioutil.ReadFile(config)
	if err != nil {
		return benchmarkConfig, err
	}

	err = json.Unmarshal(configBytes, benchmarkConfig)
	if err != nil {
		return benchmarkConfig, err
	}

	return benchmarkConfig, err
}

func prepBenchmark(config BenchmarkConfig) ([]byte, error) {
	scriptName := []string{BENCHMARK_PREPARE_SCRIPT}
	numCommands := []string{strconv.Itoa(len(config.Cmds))}
	olPath := []string{config.OlPath}
	registryMachines := config.RegistryMachines
	params := append(scriptName, numCommands...)
	params = append(params, olPath...)
	params = append(params, registryMachines...)
	//fmt.Println(exec.Command("/bin/sh", "ls").Output())
	return exec.Command("/bin/bash", params...).CombinedOutput()
}

// Generate synthetic workload based on benchmark config
func genWorkload(config BenchmarkConfig) {
	numCommands := len(config.Cmds)

	// Create WorkerPool for each Command in BenchmarkConfig
	WorkerPools = make([]*WorkerPool, numCommands)
	for i := 0; i < numCommands; i++ {
		curCommand := config.Cmds[i]
		WorkerPools[i] = &WorkerPool{}
		WorkerPools[i].workerID = i
		WorkerPools[i].numWorkers = curCommand.NumThreads
		WorkerPools[i].numJobs = curCommand.TotalCmds
		WorkerPools[i].done = make(chan bool)
		WorkerPools[i].wg = new(sync.WaitGroup)
		WorkerPools[i].chls = make(chan PerfMetrics)
		WorkerPools[i].jobs = make(chan Job, curCommand.TotalCmds)
	}

	url := makeUrl(config.Host, config.Port)
	for i := 0; i < numCommands; i++ {
		// Create Job template
		job := &Job{}
		job.url = url + RUN_LAMBDA_BASE + LAMBDA_BASE + strconv.Itoa(i+1)
		job.cmds = config.Cmds[i].Cmd
		job.lambda = LAMBDA_BASE + strconv.Itoa(i+1)
		addLambdaRequests(WorkerPools[i], job)
	}

	runWorkload()
	aggregateMetrics()
}

func makeUrl(host string, port int) string {
	return "http://" + host + ":" + strconv.Itoa(port)
}

// Adds new lambda request to a worker pool, non-blocking call
func addLambdaRequests(pool *WorkerPool, jobTemplate *Job) {
	for i := 0; i < pool.numJobs; i++ {
		job := jobTemplate
		job.jobID = i
		pool.jobs <- *job
	}
	close(pool.jobs)
}

func runWorkload() {
	for i := 0; i < len(WorkerPools); i++ {
		curWorkerPool := WorkerPools[i]
		for worker := 0; worker < curWorkerPool.numWorkers; worker++ {
			curWorkerPool.wg.Add(1)
			go cmdWorker(curWorkerPool.jobs, curWorkerPool.wg)
		}
	}
	for i := 0; i < len(WorkerPools); i++ {
		WorkerPools[i].wg.Wait()
	}
}

func cmdWorker(jobs chan Job, wg*sync.WaitGroup) int64 {
	defer wg.Done()
	for job := range jobs {
		perfMetrics := PerfMetrics{job.jobID,job.lambda,singleLambdaRequest(job)}
		//fmt.Println(perfMetrics)
		Metrics = append(Metrics, perfMetrics)
	}

	return 0
}

func simpleTest(job Job) int64 {
	time.Sleep(time.Second)
	fmt.Printf("%s", job.cmds)
	return int64(rand.Intn(10))
}

func singleLambdaRequest(job Job) int64 {
	start_time := time.Now().UnixNano()
	fmt.Println(job.url)
	resp, err := http.Post(job.url, "text/plain", bytes.NewBufferString(job.cmds))
	if err != nil {
		log.Fatalln(err)
		fmt.Println(err)
	}
	body, err :=ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	err = resp.Body.Close()
	if err!=nil {
		fmt.Println(err)
	}
	end_time := time.Now().UnixNano()
	return (end_time - start_time)/1000000
}

func aggregateMetrics() {
	//fmt.Println(Metrics)
	metricsMap := genMetricsMap()
	for lambda, metrics := range metricsMap {
		var mean = 0.0
		var std = 0.0

		for _, v := range metrics {
			mean += float64(v)
		}
		mean = mean / float64(len(metrics))
		for _, v := range metrics {
			std += math.Pow(float64(v)-mean, 2)
		}
		std = math.Sqrt(std/float64(len(metrics)-1))
		fmt.Printf("%v has average run time: %f ms, with standard deviation: %f ms \n", lambda, mean, std)
	}
}

func genMetricsMap() map[string]map[int]int64 {
	metricsMap := make(map[string]map[int]int64)
	for _, metric := range Metrics {
		jobID := metric.jobID
		lambda := metric.lambda
		runtime := metric.runTime

		if metricsMap[lambda] == nil {
			metricsMap[lambda] = make(map[int]int64)
		}
		metricsMap[lambda][jobID] = runtime
	}
	return metricsMap
}


func main(){
	cli.CommandHelpTemplate = `NAME:
   {{.HelpName}} - {{if .Description}}{{.Description}}{{else}}{{.Usage}}{{end}}
USAGE:
   {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} command{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}
COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}
{{end}}{{if .VisibleFlags}}
OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
`
	app := cli.NewApp()
	app.Usage = "Benchmark tool for Open-Lambda"
	app.UsageText = "benchmark COMMAND [ARG...]"
	app.ArgsUsage = "ArgsUsage"
	app.EnableBashCompletion = true
	app.HideVersion = true
	app.Commands = []cli.Command{
		{
			Name:      "test",
			Usage:     "test cluster",
			UsageText: "benchmark test --config=CONFIG",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "config, c",
					Usage: "Load benchmark config file from FILE",
				},
			},
			Action: startBenchmark,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

