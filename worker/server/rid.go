// OpenLambda RID
// A simple application that can be used
// to get some basic system information or send it over http
package server

import ("fmt"; "net/http"; "os/exec"; "log"; "syscall"; "strconv"; "strings")

const ( FULL_NAME = "openlambda-rid" )

var s syscall.Sysinfo_t

func nproc() uint64 {
	data, err := exec.Command("nproc").Output()
	if err != nil {
		return 0
	}

	str := string(data)
	pure_str := strings.Split(str, "\n")[0];

	ret_a, err_a := strconv.Atoi(string(pure_str))

	if err_a != nil {
		return 0
	}

	return uint64(ret_a)
}

func cpufree() string {
	data, err := exec.Command("statgrab", "-p", "-u", "cpu.idle").Output()
	if err != nil {
		log.Fatal(FULL_NAME + ": An error occurred while fetching CPU time stats. Maybe statgrab is not installed?")
	}

	return string(data)
}

func sysinf_update() {
	if syscall.Sysinfo(&s) != nil {
		log.Fatal(FULL_NAME + ": An error has occurred while attempting to gather system information");
	}
}

func freeMem() uint64 {
	return s.Freeram
}

/* RID Http Handler
 * -
 * RID gives information about the machine that is currently hosting this
 * instance of OpenLambda.
 */
type RidHttpHandler struct {
}

func (rid * RidHttpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	param := req.Form.Get("query")
	sysinf_update()

	if len(param) == 0 {
		fmt.Fprintf(resp, FULL_NAME + " usage\n")
		fmt.Fprintf(resp, "GET parameter: query = m | p | i \n")
		fmt.Fprintf(resp, "\tm - free memory info (bytes), p - logical processing unit count, i - idle cpu by percentage\n");
	}

	for i := 0; i < len(param); i++ {
		if param[i] == 'm' {
			fmt.Fprintf(resp, "%d\n", freeMem())
		}

		if param[i] == 'p' {
			fmt.Fprintf(resp, "%d\n", nproc())
		}

		if param[i] == 'i' {
			fmt.Fprintf(resp, "%s", cpufree()) // statgrab places an implicit newline
		}
	}
}
