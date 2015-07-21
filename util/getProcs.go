package util

import (
	"os"
	"fmt"
	"net"
	"bufio"
	"strings"
	"hash/fnv"
)

func GetProcs(confFile string, prnt bool) map[uint32]string {
	fi, err := os.Open(confFile)
	if err != nil {
		fmt.Println("Could not open file %v.\n", confFile)
		return nil
	}

	defer fi.Close()

	h := fnv.New32a()

	addrs := make(map[uint32]string)

	scanner := bufio.NewScanner(fi)
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		_,err = net.ResolveTCPAddr("tcp", s)
		if err != nil {
			fmt.Println("Could not parse address: ", s)
			return nil
		}
		
		h.Write([]byte(s))
		id := h.Sum32()
		addrs[id] = s
		
		if prnt {fmt.Printf("ID %v Addr %v\n",id,s)}
		h.Reset()
	}
	return addrs
}