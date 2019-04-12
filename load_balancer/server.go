package load_balancer

import (
	"strconv"
	"math"
	"sync"
)

type Server struct {
    Name string
    Scheme string
    Host string 
	Port int
	MaxConn int
    CPUUsage float64 `json:"-"`
    MemUsage float64 `json:"-"`
    Connections int `json:"-"`
    Connections int `json:"-"`
	ConnLock *sync.Mutex `json:"-"`
}

// TODO: design a better formula
func (server Server) GetLoadByConn() float64 {
	load := float64(server.Connections) / float64(server.MaxConn)
	if server.MaxConn == -1 {
		load = float64(server.Connections) / float64(math.MaxInt64)
	} 
	return load
}

func (server Server) GetLoad(loadFormula string) float64 {
	var load float64
	switch loadFormula {
	case "AverageResource":
		load = 0.5 * server.MemUsage + 0.5 * server.CPUUsage
	case "AverageResourceConnections":
		load = (server.MemUsage + server.CPUUsage + server.GetLoadByConn()) / 3.0 
	case "DominantResource":
		load = math.Max(server.MemUsage, server.CPUUsage)
	case "DominantResourceConnections":
		load = math.Max(math.Max(server.MemUsage, server.CPUUsage), server.GetLoadByConn())
	case "Connections":
		load = server.GetLoadByConn()
	}
	return load
}

func (server Server) Url() string {
  return server.Scheme + "://" + server.Host + ":" + strconv.Itoa(server.Port);
}
