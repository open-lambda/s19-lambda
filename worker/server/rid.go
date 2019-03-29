// OpenLambda RID
// A simple application that can be used
// to get some basic system information or send it over http
package server

import (
	"fmt"
	"net/http"
	"os/exec"
	"log"
	"syscall"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/cpu"
)

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

/* cpuusage
 * Returns one value
 * return value: percentage of CPU usage
 */
func cpuusage() float64 {
	v, _ := cpu.Percent(0, false)
	return v[0]
}

/* cpufree
 * Returns one value
 * return value: percentage of idle CPU
 */
func cpufree() float64 {
//	data, err := exec.Command("statgrab", "-p", "-u", "cpu.idle").Output()
//	if err != nil {
//		log.Fatal(FULL_NAME + ": An error occurred while fetching CPU time stats. Maybe statgrab is not installed?")
//	}
	// TODO remove after testing
//	return string(data)

	return  100 - cpuusage()
}


func sysinf_update() {
	if syscall.Sysinfo(&s) != nil {
		log.Fatal(FULL_NAME + ": An error has occurred while attempting to gather system information");
	}
}

/* memusefree
 * Returns two values
 * First return value: total memory available 
 * Second return value : free memory available
 */
func mem_allfree() (int, int) {
	v, _ := mem.VirtualMemory()
	return int(v.Total), int(v.Free)
}

/* RID Http Handler
 * -
 * RID gives information about the machine that is currently hosting this
 * instance of OpenLambda.
 */
type RidHttpHandler struct {
}

var hack int 

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
			t1, t2 := mem_allfree()
			fmt.Fprintf(resp, "%d\n", t2)
			hack = t1
		}

		if param[i] == 'p' {
			fmt.Fprintf(resp, "%d\n", nproc())
		}

		if param[i] == 'i' {
			fmt.Fprintf(resp, "%d\n", cpufree())
		}
	}
}
