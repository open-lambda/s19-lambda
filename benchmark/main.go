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
	Cmd        		string
	NumThreads 		int
	CmdPerLambda  	int
	NumLambdas		int
}

type WorkerPool struct{
	workerID		int
	numWorkers		int
	numLambdas		int
	jobPerLambda	int
	done			chan bool
	wg				*sync.WaitGroup
	chls			chan PerfMetrics
	jobs			chan Job
}

type Job struct{
	jobID		int
	url			string
	lambda 		string
	lambdaID	int
	cmds 		string
}

type PerfMetrics struct{
	jobID		int
	lambda		string
	lambdaID	int
	runTime 	int64
}


var LAMBDA_BASE = "lambda-"
var BENCHMARK_PREPARE_SCRIPT = "/mnt/lambda_scheduler/s19-lambda/benchmark/benchmark_prepare.sh"
var RUN_LAMBDA_BASE = "/runLambda/"
var lambdaCounter = 1

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

	cmdLineOut, err := prepBenchmark(*benchmarkConfig)
	fmt.Printf("%s", cmdLineOut)
	if err != nil {
		fmt.Println("Failed to execute prepare_benchmark.sh")
		log.Fatalln(err)
		return err
	}

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
	totalLambdas := 0
	for i := 0; i < len(config.Cmds); i++ {
		totalLambdas += config.Cmds[i].NumLambdas
	}
	numLambdas := []string{strconv.Itoa(totalLambdas)}
	olPath := []string{config.OlPath}
	registryMachines := config.RegistryMachines
	params := append(scriptName, numLambdas...)
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
		totalCmds := curCommand.NumLambdas * curCommand.CmdPerLambda
		WorkerPools[i] = &WorkerPool{}
		WorkerPools[i].workerID = i
		WorkerPools[i].numWorkers = curCommand.NumThreads
		WorkerPools[i].numLambdas = curCommand.NumLambdas
		WorkerPools[i].jobPerLambda = curCommand.CmdPerLambda
		WorkerPools[i].done = make(chan bool)
		WorkerPools[i].wg = new(sync.WaitGroup)
		WorkerPools[i].chls = make(chan PerfMetrics)
		WorkerPools[i].jobs = make(chan Job, totalCmds)
	}

	baseUrl := "http://" + config.Host + ":" + strconv.Itoa(config.Port)
	for i := 0; i < numCommands; i++ {
		// Create Job template
		job := &Job{}
		// create base job url
		job.url = baseUrl
		job.cmds = config.Cmds[i].Cmd

		addLambdaRequests(WorkerPools[i], job)
	}

	runWorkload()
	aggregateMetrics()
}

func makeUrl(baseUrl string, lambda string) string {
	return baseUrl + RUN_LAMBDA_BASE + lambda
}

// Adds new lambda request to a worker pool, non-blocking call
func addLambdaRequests(pool *WorkerPool, jobTemplate *Job) {
	var jobID = 0

	for i:= 0; i < pool.jobPerLambda; i++ {
		for j := 0; j < pool.numLambdas; j++ {
			job := &Job{}
			job.lambdaID = lambdaCounter + j
			job.jobID = jobID
			jobID++
			job.lambda = LAMBDA_BASE + strconv.Itoa(job.lambdaID)
			job.url = makeUrl(jobTemplate.url, job.lambda)
			job.cmds = jobTemplate.cmds
			pool.jobs <- *job
		}
	}
	lambdaCounter += pool.numLambdas

	//for i := 0; i < pool.numLambdas; i++ {
	//	for j:= 0; j < pool.jobPerLambda; j++ {
	//		job := &Job{}
	//		job.lambdaID = lambdaCounter
	//		job.jobID = jobID
	//		jobID++
	//		job.lambda = LAMBDA_BASE + strconv.Itoa(job.lambdaID)
	//		job.url = makeUrl(jobTemplate.url, job.lambda)
	//		job.cmds = jobTemplate.cmds
	//		pool.jobs <- *job
	//	}
	//	lambdaCounter++
	//}

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
		perfMetrics := PerfMetrics{job.jobID,job.lambda,job.lambdaID,singleLambdaRequest(job)}
		//fmt.Println(perfMetrics)
		Metrics = append(Metrics, perfMetrics)
	}

	return 0
}

func simpleTest(job Job) int64 {
	time.Sleep(time.Second)
	fmt.Println(job)
	return int64(rand.Intn(10))
}

func singleLambdaRequest(job Job) int64 {
	start_time := time.Now().UnixNano()
	resp, err := http.Post(job.url, "application/json", bytes.NewBufferString(job.cmds))
	//fmt.Println(bytes.NewBufferString(job.cmds))
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

