package load_balancer

import (
	"fmt"
	"strings"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"encoding/json"
)

const configName string = "config.yml"

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
		proxy.Port = 8079
	}

	if proxy.Scheme == "" {
		proxy.Scheme = "http"
	}

	if proxy.Servers == nil {
		proxy.Servers = []Server{}
	}

	if proxy.LoadHigh == nil{
		proxy.LoadHigh = 0.4
	}

	if proxy.LoadLow == nil{
		proxy.LoadLow = 0.3
	}

	if proxy.RequestServerMap == nil{
		proxy.RequestServerMap = make(map[string]*Server)
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


