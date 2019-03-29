package load_balancer

import "strconv"

type Server struct {
    Name string
    Scheme string
    Host string
    Port int
    Connections int
    Cpu float64
    MemPercent float64
}

// TODO: design a better formula
func (server Server) GetLoad() float64 {
    return server.MemPercent + float64(server.Connections)/float64(1000)
}

func (server Server) Url() string {
  return server.Scheme + "://" + server.Host + ":" + strconv.Itoa(server.Port);
}
