package load_balancer

import (
	"fmt"
	"strings"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"encoding/json"
	"sync"
)

const configName string = "config.yml"
const defaultServerMaxConn int = -1

func validation(condition bool, errorMessage string) string {
	if condition {
		return errorMessage
	} else {
		return ""
	}
}

func removeEmpty(errors []string) []string {
	var filtered = []string{}
	for _,e := range errors {
		if e != "" {
			filtered = append(filtered, e)
		}
	}

	return filtered
}

func generateValidationErrors(proxy *Proxy) []string {
	return removeEmpty([]string{
		validation(
			proxy.Host == "",
			"the 'host' field cannot be blank",
		),
		validation(
			proxy.Port == 0,
			"the 'port' field cannot be blank",
		),
		/*
		validation(
			len(proxy.Servers) == 0,
			"the config must specify at least 1 server",
		),
		*/
		validation(
			proxy.Scheme != "http" && proxy.Scheme != "https",
			"the proxy scheme must be either 'http' or 'https'",
		),
	})
}

func validateFields(proxy *Proxy) error {
	var errors = generateValidationErrors(proxy)

	if(len(errors) == 0) {
		return nil
	} else {
		return fmt.Errorf(strings.Join(errors, ", "))
	}
}

func SetDefaultValues(proxy *Proxy) {
	if proxy.Host == "" {
		proxy.Host = "localhost"
	}

	if proxy.Port == 0 {
		proxy.Port = 7079
	}

	if proxy.Scheme == "" {
		proxy.Scheme = "http"
	}

	if proxy.Policy == "" {
		proxy.Policy = "LARD"
	}

	if proxy.LoadFormula == "" {
		proxy.LoadFormula = "Connections"
	}

	if proxy.Servers == nil {
		proxy.Servers = []Server{}
	}

	for i, _ := range proxy.Servers {
		server := &proxy.Servers[i]
		// when user DIT NOT SPECIFY max num of connections, use default value server.MaxConn = 2000
		if server.MaxConn == 0 { 
			server.MaxConn = defaultServerMaxConn
		} 
	}

	if proxy.LoadHigh == 0.0 {
		proxy.LoadHigh = 0.4
	}

	if proxy.LoadLow == 0.0 {
		proxy.LoadLow = 0.2
	}

	if proxy.RequestServerMap == nil {
		proxy.RequestServerMap = make(map[string]*Server)
	}

	if proxy.MaxConn == 0 {
		if len(proxy.Servers) == 0 {
			proxy.MaxConn = -1
		} else {
			for _, server := range proxy.Servers {
				// when user specify no size limit for any worker queue, 
				// server queue also will not have size limit
				if server.MaxConn == -1 {
					proxy.MaxConn = -1
					break
				}
				proxy.MaxConn += server.MaxConn
			}	
		}
	}
}

func ReadConfig(config_file string) (*Proxy, error) {
	proxy := &Proxy{}

	file, err := ioutil.ReadFile(config_file)
	if err != nil {
		return proxy, err
	}

	if strings.HasSuffix(config_file, ".yml") {
		err = yaml.Unmarshal(file, proxy)
	} else {
		err = json.Unmarshal(file, proxy)
	}

	if err != nil {
		return proxy, err
	}

	SetDefaultValues(proxy)

	lock := &sync.Mutex{}
	proxy.ConnCond = sync.NewCond(lock)

	for i, _ := range proxy.Servers {
		server := &proxy.Servers[i]
		server.ConnLock = &sync.Mutex{}
	}

	err = validateFields(proxy)
	if err != nil {
		return proxy, err
	}

	return proxy, nil
}

func DumpStr(proxy *Proxy) string {
	s, err := json.MarshalIndent(proxy, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

func SaveConfig(proxy *Proxy, path string) error {
	s, err := json.MarshalIndent(proxy, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, s, 0644)
}


