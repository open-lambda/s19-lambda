// WiscEdge RID
// A simple application that can be used
// to get some basic system information or send it over http
package main

import ("os/exec"; "os"; "fmt"; "log"; "syscall"; "strconv"; "strings")

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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: wiscedge-rid [-m] [-p]")
		fmt.Println("INTERACTIVE USAGE")
		fmt.Println("[-m] - get free memory in bytes")
		fmt.Println("[-p] - get processor unit count")
		fmt.Println("If multiple flags are specified, then data will be reported in the order of the flags")
		fmt.Println("HTTP SERVER USAGE")
		fmt.Println("[-d [PORT]] - start as HTTP server on port PORT")
		fmt.Println("Once the http server is running, simply send the letters of the same flags in interative mode as the query parameter (such as query=mp) for the corresponding information")
		fmt.Println("Context path: /info")
		return
	}

	sysinf_update()

	if os.Args[1] == "-d" {

		if len(os.Args) < 3 {
			log.Fatal("wiscedge-rid: insufficient arguments, must specify port")
		} else {
			createHttpd(os.Args[2]);
		}

	} else {
		for i := 1; i < len(os.Args); i++ {
			switch os.Args[i] {
				case "-m":
					fmt.Printf("%d\n", mem())
				case "-p":
					fmt.Printf("%d\n", nproc());
			}
		}
	}
}
