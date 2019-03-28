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

	workerServer "github.com/open-lambda/open-lambda/worker/server"
)

type Proxy struct {
	Host string
	Port int
	Scheme string
	Servers []Server
}

func (proxy Proxy) origin() string {
	return (proxy.Scheme + "://" + proxy.Host + ":" + strconv.Itoa(proxy.Port));
}

// TODO: This crashes if we define no servers in our config
func (proxy Proxy)chooseServer(ignoreList []string) *Server {
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

var RRServerIdx int = 0

func (proxy Proxy)roundRobinChooseServer(ignoreList []string) *Server {
	if len(proxy.Servers) == 0 {
		return nil
	}
	firstAttempt := true
	nextServerIndex := (RRServerIdx + 1) % len(proxy.Servers)
	for firstAttempt || (nextServerIndex != (RRServerIdx + 1)) {
		server := proxy.Servers[nextServerIndex]
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
		return &server
	}
	return nil
}

func (proxy Proxy)ReverseProxy(w http.ResponseWriter, r *http.Request, server Server) (int, error){
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
	LogInfo("Total Memory: " + strconv.Itoa(respStruct.TotalMem))
	LogInfo("Free Memory: " + strconv.Itoa(respStruct.FreeMem))
	LogInfo("CPU Usage: " + fmt.Sprintf("%f", respStruct.CPUUsage))

	if err != nil {
		LogErr("Proxy: Failed to read response body")
		http.NotFound(w, r)
		return 0, err
	}

	// buffer := bytes.NewBuffer(bodyBytes)
	for k, v := range respStruct.ResponseHeader {
		w.Header().Set(k, strings.Join(v, ";"))
	}

	w.WriteHeader(respStruct.ResponseCode)

	// io.Copy(w, buffer)
	// realBody, _ := ioutil.ReadAll(respStruct.ResponseBody)
	w.Write(respStruct.ResponseBody)
	fmt.Print(w.Header())
	return respStruct.ResponseCode, nil
}

func (proxy Proxy)attemptServers(w http.ResponseWriter, r *http.Request, ignoreList []string) {
	if float64(len(ignoreList)) >= math.Min(float64(3), float64(len(proxy.Servers))) {
		LogErr("Failed to find server for request")
		http.NotFound(w, r)
		return
	}

	// var server = proxy.chooseServer(ignoreList)
	var server = proxy.roundRobinChooseServer(ignoreList)
	if server == nil {
		LogErr("Proxy: Could not find an available server at this time")
		http.NotFound(w, r)
		return
	}

	LogInfo("Got request: " + r.RequestURI)
	LogInfo("Sending to server: " + server.Name)

	server.Connections += 1
	_, err := proxy.ReverseProxy(w, r, *server)
	server.Connections -= 1

	if err != nil && strings.Contains(err.Error(), "connection refused") {
		LogWarn("Server did not respond: " + server.Name)
		proxy.attemptServers(w, r, append(ignoreList, server.Name))
		return
	}

	LogInfo("Responded to request successfuly")
}

func (proxy Proxy)handler(w http.ResponseWriter, r *http.Request) {
	proxy.attemptServers(w, r, []string{})
}

func (proxy Proxy)statusHandler(w http.ResponseWriter, r *http.Request) {
	LogInfo("Receive request to " + r.URL.Path)

	wbody := []byte("ready\n")
	if _, err := w.Write(wbody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
