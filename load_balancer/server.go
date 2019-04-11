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
    Cpu float64
    MemPercent float64
    Connections int `json:"-"`
	MaxConn int
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
		load = 0.5 * server.MemPercent + 0.5 * server.Cpu
	case "AverageResourceConnections":
		load = (server.MemPercent + server.Cpu + server.GetLoadByConn()) / 3.0 
	case "DominantResource":
		load = math.Max(server.MemPercent, server.Cpu)
	case "DominantResourceConnections":
		load = math.Max(math.Max(server.MemPercent, server.Cpu), server.GetLoadByConn())
	case "Connections":
		load = server.GetLoadByConn()
	}
	return load
}

func (server Server) Url() string {
  return server.Scheme + "://" + server.Host + ":" + strconv.Itoa(server.Port);
}
