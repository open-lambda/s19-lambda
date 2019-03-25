// OpenLambda History Module
// Binds to thie /history path and when invoked will print out a list of lambda
// handlers in a known format, by default CSV, also can be in JSON, YML, or XML

package server

import ("fmt"; "math/rand"; "net/http"; "sync"; "time")

// please, if some type of union or enum type exists in Go please replace this struct type with a union
type Ops struct {
	PEEK uint8	// PEEK mode: do a lookup, do not do any modifications 
	OPEN uint8	// OPEN mode: do a lookup, and write corresponding handler entry's present to 1
	CLOSE uint8	// CLOSE mode: do a lookup, and write corresponding handler entry's present to 0
}
var CODES = Ops{0, 1, 2} // bootleg enum, yeah!

// DO NOT MODIFY ANY OF THESE FIELDS DIRECTLY,
// Instead, go through HandlerAccess only
type LambdaHistoryHandler struct {
	logSize uint32		// how many entries to keep 
	hnames []string		// a list of handler names	
	present []bool		// does the current handler have a warmed container?
				// true if yes, false if no
	writeLock sync.Mutex	// structure write lock
}

func (hist * LambdaHistoryHandler) PrintCSVToResp(resp *http.ResponseWriter) {
	var i uint32
	var p int8
	for i = 0; i <  hist.logSize; i++ {
		var s string
		if hist.hnames[i]  == "" {
			s = "[cold]"
		} else {
			s =  hist.hnames[i]
		}

		if hist.present[i]  {
			p = 1
		} else {
			p = 0
		}

		fmt.Fprintf(*resp, "%s,%d\n", s, p)
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

// The core function responsible for maintaining history details
// This function is akin a simple lookup on the table
// Otherwise, it will create a new entry in the table and possibly evict an existing entry (using random)
// Returns: present or not present bit
func (hist * LambdaHistoryHandler) HandlerAccess(hname string, code uint8) uint8 {
	var ind uint32
	var found bool
	// TODO: for better performance, change to hash map string lookup is slow 
	for ind = 0; ind < hist.logSize; ind++ {

		// found!
		if(hist.hnames[ind] == hname) {
			found = true
			break
		}

		// reached a cold spot!
		if(hist.hnames[ind] == "") {
			hist.hnames[ind] = hname
			found = true
			break
		}
	}

	// If not found and not cold, then go on and evict a random entry
	if !found {
		rand.Seed(time.Now().UnixNano())
		ran := rand.Uint32() % hist.logSize
		hist.hnames[ran] = hname
		ind = ran
	}

	var p uint8 = 0

	if code == CODES.OPEN {
		p = 1
		hist.present[ind] = true
	}

	if code == CODES.CLOSE {
		p = 0
		hist.present[ind] = false
	}

	if code == CODES.PEEK {
		if hist.present[ind] {
			p = 1
		} else {
			p = 0
		}
	}

	return p
}
