package load_balancer

import (
	"strconv"
	"math"
	"sync"
	"log"
)

type Server struct {
    Name string
    Scheme string
    Host string 
	Port int
	MaxConn int
    CPUUsage float64 `json:"-"`
    MemUsage float64 `json:"-"`
	LoadMultiplier float64 
    Connections int `json:"-"`
	ConnLock *sync.Mutex `json:"-"`
    ForkServerMetaList []*ForkServerMeta
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
	case "LoadMultiplier":
		load = server.LoadMultiplier
	}
	return load
}

func (server Server) Url() string {
  return server.Scheme + "://" + server.Host + ":" + strconv.Itoa(server.Port);
}


func (server Server) Match(imports []string) int {
	fsmList := server.ForkServerMetaList
	best_score := -1
	log.Printf("fmsList len: %d, cap: %d", len(fsmList), cap(fsmList))
	for i := 0; i < len(fsmList); i++ {
		matched := 0
		for j := 0; j < len(imports); j++ {
			if fsmList[i].Imports[imports[j]] {
				matched += 1
			}
		}

		log.Printf("matched: %d ,listLen: %d", matched, len(fsmList[i].Imports))
		// constrain to subset
		if matched > best_score && len(fsmList[i].Imports) <= matched {
			best_score = matched
		}
	}
	log.Printf("Best_score: %d", best_score)
	return best_score
}

func (server Server) AddNewForkServerMeta(parent *ForkServerMeta, imports []string) {
	//add lock??????
	fsm := &ForkServerMeta{
		Imports:  make(map[string]bool),
		Hits:     0.0,
	}

	for key, val := range parent.Imports {
		fsm.Imports[key] = val
	}
	for k := 0; k < len(imports); k++ {
		fsm.Imports[imports[k]] = true
	}

	leastHitFSMIndex := server.getLeastHitForkServerMetaIndex()

	//recycle?
	server.ForkServerMetaList[leastHitFSMIndex] = fsm
	server.HitForkServerMeta(fsm)
	return
}

func (server Server) getLeastHitForkServerMetaIndex() int {
	fsmList := server.ForkServerMetaList
	if len(fsmList) < cap(fsmList) {
		return len(fsmList)
	}
	leastHitNum := fsmList[0].Hits
	leastHitIndex := 0
	for i, fsm := range fsmList {
		if leastHitNum > fsm.Hits {
			leastHitNum = fsm.Hits
			leastHitIndex = i
		}
	}
	
	return leastHitIndex
}

func (server Server) HitForkServerMeta(hitForkServerMeta *ForkServerMeta) {
	for _, fsm := range server.ForkServerMetaList {
		fsm.Hit(hitForkServerMeta == fsm) // correct comparasion????????????????
	}
}
