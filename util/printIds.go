package main

import (
	"os"
	"fmt"
	"net"
	"flag"
	"bufio"
	"strings"
	"hash/fnv"

	//"github.com/relab/smartMerge/rpc"
)

var (
	confFile = flag.String("conf","config", "the config file, a list of host:port addresses.")
	prnt = flag.Bool("print", true, "print to screen")
)

func main() {
	flag.Parse()
	fi, err := os.Open(*confFile)
	if err != nil {
		fmt.Println("Could not open file %v.\n", confFile)
		return
	}

	defer fi.Close()

	h := fnv.New32a()

	addr := make([]string, 0,10)
	ids := make([]uint32, 0,10)

	scanner := bufio.NewScanner(fi)
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		_,err = net.ResolveTCPAddr("tcp", s)
		if err != nil {
			fmt.Println("Could not parse address: ", s)
			return
		}
		addr = append(addr, s)
		h.Write([]byte(s))
		id := h.Sum32()
		ids = append(ids, id)
		if *prnt {fmt.Printf("%v; %v\n",id,s)}
		h.Reset()
	}
}
