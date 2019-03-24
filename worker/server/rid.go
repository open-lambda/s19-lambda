// WiscEdge RID
// A simple application that can be used
// to get some basic system information or send it over http
package server

import ("os/exec"; "log"; "syscall"; "strconv"; "strings")

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
		log.Fatal("wiscedge-rid: An error has occurred while attempting to gather system information");
	}
}

func mem() uint64 {
	return s.Freeram
}
