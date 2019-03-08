package balancer

import (
    "net/http"
    "strconv"
)

func main(config_file string) {
    LogInfo("Spinning up load balancer...")
    LogInfo("Reading config file...")
    proxy, err := ReadConfig(config_file)

    if err != nil {
      LogErr("An error occurred while trying to parse config.yml")
      LogErrAndCrash(err.Error())
    }
    http.HandleFunc("/", proxy.handler)
    err = http.ListenAndServe(":" + strconv.Itoa(proxy.Port), nil)
    if err != nil {
        LogErr("Failed to bind to port " + strconv.Itoa(proxy.Port))
        LogErrAndCrash("Make sure the port is available and not reserved")
    }
    LogInfo("Listening to requests on port: " + strconv.Itoa(proxy.Port))
}

func Main(config_file string) {
    main(config_file)
}
