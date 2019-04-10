package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
	Cmd        string
	NumThreads int
	TotalCmds  int
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
	url			string
	lambda 		string
	cmds 		string
}

type PerfMetrics struct{
	runTime 	int64
}


var LAMBDA_BASE = "lambda-"
var BENCHMARK_PREPARE_SCRIPT = "benchmark_prepare.sh"

// WorkerPools consists a list of WorkerPool, one WorkerPool for each
// Command in the corresponding config file.
var WorkerPools []*WorkerPool

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
	if err != nil {
		fmt.Println("Failed to execute run_benchmark.sh")
		log.Fatalln(err)
		return err
	}
	log.Printf("%s", cmdLineOut)

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
	return exec.Command("/bin/sh", params...).Output()
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
		job.url = url
		job.cmds = config.Cmds[i].Cmd
		job.lambda = LAMBDA_BASE + string(i)
		addLambdaRequests(WorkerPools[i], job)
	}

	runWorkload()
}

func makeUrl(host string, port int) string {
	return "http://" + host + ":" + strconv.Itoa(port)
}

// Adds new lambda request to a worker pool, non-blocking call
func addLambdaRequests(pool *WorkerPool, job *Job) {
	for i := 0; i < pool.numJobs; i++ {
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
		perfMetrics := PerfMetrics{singleLambdaRequest(job)}
		fmt.Printf("%d", perfMetrics)
	}

	return 0
}

func simpleTest(job Job) int64 {
	time.Sleep(time.Second)
	fmt.Printf("%s", job.cmds)
	return 0
}

func singleLambdaRequest(job Job) int64 {
	start_time := time.Now().UnixNano()
	_, err := http.Post(job.url, "text/plain", bytes.NewBufferString(job.cmds))
	if err != nil {
		log.Fatalln(err)
	}
	end_time := time.Now().UnixNano()
	return end_time - start_time
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