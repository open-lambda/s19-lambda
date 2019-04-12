package load_balancer

import (
	"net/http"
	"net/url"
	"io/ioutil"
	// "bytes"
	"strings"
	"strconv"
	"math"
	"encoding/json"
	"fmt"
	"sync"

	workerServer "github.com/open-lambda/s19-lambda/worker/server"
)

type Proxy struct {
	Host string
	Port int
	Scheme string
	Policy string
	LoadFormula string
	Servers []Server
	RequestServerMap map[string]*Server
	MapLock sync.RWMutex `json:"-"`
	LoadHigh float64
	LoadLow float64
	Connections int `json:"-"`
	MaxConn int // max num of connections for the whole system
	ConnCond *sync.Cond `json:"-"` // condional variable for connections
}

func (proxy *Proxy) origin() string {
	return (proxy.Scheme + "://" + proxy.Host + ":" + strconv.Itoa(proxy.Port));
}

// TODO: This crashes if we define no servers in our config
func (proxy *Proxy)chooseServer(ignoreList []string) *Server {
	var min = -1
	var minIndex = 0
	for index,server := range proxy.Servers {
		var skip = false
		for _, ignore := range ignoreList {
			if(ignore == server.Name){
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		var conn = server.Connections
		if min == -1 {
			min = conn
			minIndex = index
		}else if(conn < min){
			min = conn
			minIndex = index
		}
	}

	return &proxy.Servers[minIndex]
}

var LeastLoadMutex = &sync.Mutex{}
var LeastLoadIdx int = 0

var RRServerIdx int = 0

func (proxy *Proxy)roundRobinChooseServer(ignoreList []string) *Server {
	if len(proxy.Servers) == 0 {
		return nil
	}
	firstAttempt := true
	nextServerIndex := (RRServerIdx + 1) % len(proxy.Servers)
	for firstAttempt || (nextServerIndex != (RRServerIdx + 1)) {
		server := &proxy.Servers[nextServerIndex]
		shouldIgnore := false
		for _, ignoreServerName := range ignoreList {
			if server.Name == ignoreServerName {
				shouldIgnore = true
				break
			}
		}
		if shouldIgnore {
			firstAttempt = false
			nextServerIndex = (nextServerIndex + 1) % len(proxy.Servers)
			continue
		}
		RRServerIdx = nextServerIndex
		server.ConnLock.Lock()
		server.Connections += 1
		server.ConnLock.Unlock()
		return server
	}
	return nil
}

func (proxy *Proxy)lardChooseServer(ignoreList []string, r *http.Request) *Server {
	var path = r.URL.Path
	proxy.MapLock.RLock()
	targetServer, ok := proxy.RequestServerMap[path]
	proxy.MapLock.RUnlock()
	if (ok) {
		// impose a limit on targetServer's queue size
		if(targetServer.GetLoad(proxy.LoadFormula) >= proxy.LoadHigh && proxy.getLeastLoad(false).GetLoad(proxy.LoadFormula) < proxy.LoadLow || targetServer.GetLoad(proxy.LoadFormula) >= 2 * proxy.LoadHigh || targetServer.MaxConn != -1 && targetServer.Connections >= targetServer.MaxConn){
			targetServer = proxy.getLeastLoad(true)
		}
		shouldIgnore := false
		for _, ignoreServerName := range ignoreList {
			if targetServer.Name == ignoreServerName {
				shouldIgnore = true
				break
			}
		}
		if shouldIgnore == true {
			targetServer = nil
		}
	} else {
		targetServer = proxy.getLeastLoad(true)
	}
	if targetServer != nil {
		targetServer.Connections += 1
		proxy.MapLock.Lock()
		proxy.RequestServerMap[path] = targetServer
		proxy.MapLock.Unlock()
	}
	return targetServer
}

func (proxy *Proxy) getLeastLoad(moveIndex bool) *Server{
	var targetServer *Server = nil
	var minLoad = 1.0
	var startIdx = 0
	if moveIndex {
		LeastLoadMutex.Lock()
		startIdx = LeastLoadIdx
		LeastLoadIdx = (LeastLoadIdx + 1) % len(proxy.Servers)
		LeastLoadMutex.Unlock()
	}
	for i:=0; i < len(proxy.Servers); i++{
		
		server := &proxy.Servers[(startIdx + i) % len(proxy.Servers)]
		// enforce a limit on worker's queue size
		if server.MaxConn != -1 && server.Connections > server.MaxConn {
			continue
		}
		var curLoad = server.GetLoad(proxy.LoadFormula)
		if curLoad < minLoad {
			minLoad = curLoad
			targetServer = server
		}
	}
	return targetServer
}

func (proxy *Proxy)ReverseProxy(w http.ResponseWriter, r *http.Request, server *Server) (int, error){

	u, err := url.Parse(server.Url() + r.RequestURI)
	if err != nil {
		LogErrAndCrash(err.Error())
	}

	r.URL = u
	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("Origin", proxy.origin())
	r.Host = server.Url()
	r.RequestURI = ""


	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// TODO: If the server doesn't respond, try a new web server
	// We could return a status code from this function and let the handler try passing the request to a new server.
	resp, err := client.Do(r)
	if err != nil {
		// For now, this is a fatal error
		// When we can fail to another webserver, this should only be a warning.
		LogErr("connection refused")
		return 0, err
	}
	LogInfo("Recieved response: " + strconv.Itoa(resp.StatusCode))

	bodyBytes, err := ioutil.ReadAll(resp.Body)

	var respStruct workerServer.HttpResp
	if err := json.Unmarshal(bodyBytes, &respStruct); err != nil {
		fmt.Print(err)
		LogErr("json unmarshal failed")
	}


	if proxy.Policy == "LARD" {
		var path = r.URL.Path
		proxy.MapLock.RLock()
		targetServer, _ := proxy.RequestServerMap[path]
		proxy.MapLock.RUnlock()
		targetServer.CPUUsage = respStruct.CPUUsage
		targetServer.MemUsage = 1.0 - float64(respStruct.FreeMem) / float64(respStruct.TotalMem)
	}

	LogInfo("Server load condition:")
	for _, server := range proxy.Servers {
		// LogInfo("Piggybacked Info: ")
		LogInfo(server.Name)
		LogInfo(fmt.Sprintf("	Memory Usage: %f%%", server.MemUsage*100))
		LogInfo(fmt.Sprintf("	CPU Usage: %f%%", server.CPUUsage))
		LogInfo(fmt.Sprintf("	Current Serving Requests: %d", server.Connections))
	}

	if err != nil {
		LogErr("Proxy: Failed to read response body")
		http.NotFound(w, r)
		return 0, err
	}

	w.WriteHeader(respStruct.ResponseCode)

	// buffer := bytes.NewBuffer(bodyBytes)
	for k, v := range respStruct.ResponseHeader {
		w.Header().Set(k, strings.Join(v, ";"))
	}

	// io.Copy(w, buffer)
	// realBody, _ := ioutil.ReadAll(respStruct.ResponseBody)
	w.Write(respStruct.ResponseBody)
	return respStruct.ResponseCode, nil
}

func (proxy *Proxy)attemptServers(w http.ResponseWriter, r *http.Request, ignoreList []string) {
	if float64(len(ignoreList)) >= math.Min(float64(3), float64(len(proxy.Servers))) {
		LogErr("Failed to find server for request")
		http.NotFound(w, r)
		return
	}

	var server *Server = nil
	switch proxy.Policy {
		case "RoundRobin":
			server = proxy.roundRobinChooseServer(ignoreList)
		case "LARD":
			server = proxy.lardChooseServer(ignoreList, r)
	}
	
	if server == nil {
		LogErr("Proxy: Could not find an available server at this time")
		http.NotFound(w, r)
		return
	}

	LogInfo("Got request: " + r.RequestURI)
	LogInfo("Sending to server: " + server.Name)

	_, err := proxy.ReverseProxy(w, r, server)

	server.ConnLock.Lock()
	server.Connections -= 1
	server.ConnLock.Unlock()

	if err != nil && strings.Contains(err.Error(), "connection refused") {
		LogWarn("Server did not respond: " + server.Name)
		proxy.attemptServers(w, r, append(ignoreList, server.Name))
		return
	}

	LogInfo("Responded to request successfuly")
}

func (proxy *Proxy)handler(w http.ResponseWriter, r *http.Request) {
	// wait on cond var when num of connections to proxy hits limit
	proxy.ConnCond.L.Lock()
	for proxy.MaxConn != -1 && proxy.Connections >= proxy.MaxConn {
		proxy.ConnCond.Wait()	
	}
	proxy.Connections += 1
	proxy.ConnCond.L.Unlock()
	
	proxy.attemptServers(w, r, []string{})
	// after served a request, decrease num of connections, and wake up all other routines waiting
	proxy.ConnCond.L.Lock()
	proxy.Connections -= 1
	proxy.ConnCond.L.Unlock()

	proxy.ConnCond.Broadcast()	
}

func (proxy *Proxy)statusHandler(w http.ResponseWriter, r *http.Request) {
	LogInfo("Receive request to " + r.URL.Path)

	wbody := []byte("ready\n")
	if _, err := w.Write(wbody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
