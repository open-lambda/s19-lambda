// OpenLambda RID
// A simple application that can be used
// to get some basic system information or send it over http
package server

import ("fmt"; "net/http"; "os/exec"; "log"; "syscall"; "strconv"; "strings")

const ( FULL_NAME = "openlambda" )

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

func sysinf_update() {
	if syscall.Sysinfo(&s) != nil {
		log.Fatal(FULL_NAME + ": An error has occurred while attempting to gather system information");
	}
}

func mem() uint64 {
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
		fmt.Fprintf(resp, "openlambda-rid usage\n")
		fmt.Fprintf(resp, "parameter: query = m | p\n")
		fmt.Fprintf(resp, "\tm - free memory info (bytes), p - logical processing unit count\n");
	}

	for i := 0; i < len(param); i++ {
		if param[i] == 'm' {
			fmt.Fprintf(resp, "%d\n", mem())
		}

		if param[i] == 'p' {
			fmt.Fprintf(resp, "%d\n", nproc())
		}

	}
}
