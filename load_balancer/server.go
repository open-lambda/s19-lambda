package balancer

import "strconv"

type Server struct {
    Name string
    Scheme string
    Host string
    Port int
    Connections int
}

func (server Server) Url() string {
  return server.Scheme + "://" + server.Host + ":" + strconv.Itoa(server.Port);
}
