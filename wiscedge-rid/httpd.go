// Wiscedge-rid Implementation
// of HTTP Server

// package server // todo: refactor as separate package
package main

import("fmt"; "net/http"; "log")

type RidHttpHandler struct {
}

type DefaultHandler struct {
}

type LocalHandler struct {

}

var rreq_name[]string;

type HistoryHandler struct {
	lru_size int	// size of the lru list
}

func (rid * LocalHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Println("Request serviced for handler: " + req.URL.Path)

	for i := (len(rreq_name) - 1) ; i > 0; i-- {
		rreq_name[i] = rreq_name[i - 1]
	}

	rreq_name[0] = req.URL.Path
}

func (rid * DefaultHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(resp, "Request Paths\n")
	fmt.Fprintf(resp, "-\n")
	fmt.Fprintf(resp, "/info?query=(m|p)(m|p)\n")
	fmt.Fprintf(resp, "* m - gives information about free memory\n")
	fmt.Fprintf(resp, "* p - gives information about total processing units\n")
	fmt.Fprintf(resp, "/history - see what jobs have been recently sent this way\n")
	fmt.Fprintf(resp, "/{handler_name} - send a lambda to a local instance\n")
}

func (rid * HistoryHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(resp, "RECENT JOBS\n")
	fmt.Fprintf(resp, "rank,handler_name\n")
	for i := 0; i < rid.lru_size; i++ {
		var s string;
		if(rreq_name[i] == "") {
			s = "[none]"
		} else {
			s = rreq_name[i]
		}

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

func createHttpd(port string) {

	http.Handle("/info", new(RidHttpHandler))
	h := HistoryHandler{0};
	h.lru_size = 16;
	rreq_name = make([]string, h.lru_size)
	http.Handle("/history", &h)
	http.Handle("/", new (LocalHandler))
	http.Handle("/help", new (DefaultHandler))
	err := http.ListenAndServe(":" + port, nil)
	if err != nil {
		log.Fatal("wiscedge-rid: failure to bind on port " + port)
	} else {
		log.Println("wiscedge-rid: listening on port: " + port)
	}
}
