// OpenLambda History Module
// Binds to thie /history path and when invoked will print out a list of lambda
// handlers in a known format, by default CSV, also can be in JSON, YML, or XML

package server

import ("fmt"; "net/http")

type LambdaHistoryHandler struct {
	logSize uint32	// how many entries to keep 
	hnames []string	// a list of handler names	
	present []bool	// does the current handler have a warmed container?
			// true if yes, false if no
}

func (hist * LambdaHistoryHandler) PrintCSVToResp(resp *http.ResponseWriter) {
	var i uint32
	for i = 0; i <  hist.logSize; i++ {
		var s string
		if hist.hnames[i]  == "" {
			s = "<cold>"
		} else {
			s =  hist.hnames[i]
		}

		fmt.Fprintf(*resp, "%s,\n", s, hist.present[i])
	}
}

func (hist * LambdaHistoryHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	hist.PrintCSVToResp(&resp)
}

func (hist * LambdaHistoryHandler) Init(size uint32) {
	hist.logSize =  size
	hist.hnames = make([]string, size)
	hist.present = make([]bool, size)
}
