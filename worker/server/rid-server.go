// Wiscedge-rid Implementation
// of HTTP Server

// package server // todo: refactor as separate package
package server

import("fmt"; "net/http")

type RidHttpHandler struct {
}

type InfoHandler struct {
}

type HistoryHandler struct {
	lru_size int	// size of the lru list
}

func (rid * HistoryHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(resp, "RECENT JOBS\n")
	fmt.Fprintf(resp, "rank,handler_name\n")
	for i := 0; i < rid.lru_size; i++ {
		var s string;
		fmt.Fprintf(resp, "%d,%s\n", i, s)
	}
}

func (rid * RidHttpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	param := req.Form.Get("query")
	sysinf_update()

	for i := 0; i < len(param); i++ {
	// Todo, refactor	
		if param[i] == 'm' {
			fmt.Fprintf(resp, "%d\n", mem())
		}

		if param[i] == 'p' {
			fmt.Fprintf(resp, "%d\n", nproc())
		}

	}
}
