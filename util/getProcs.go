package util

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"strings"
)

func GetProcs(confFile string, prnt bool) (addrs []string, ids []uint32) {
	fi, err := os.Open(confFile)
	if err != nil {
		fmt.Println("Could not open file %v.\n", confFile)
		return nil, nil
	}

	defer fi.Close()

	h := fnv.New32a()

	addrs = make([]string, 0)
	ids = make([]uint32, 0)

	scanner := bufio.NewScanner(fi)
	if prnt {
		fmt.Printf("Processes from Config file")
	}
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		_, err = net.ResolveTCPAddr("tcp", s)
		if err != nil {
			fmt.Println("Could not parse address: ", s)
			return nil, nil
		}

		h.Write([]byte(s))
		id := h.Sum32()
		addrs = append(addrs, s)
		ids = append(ids, id)

		if prnt {
			fmt.Printf("ID %v Addr %v\n", id, s)
		}
		h.Reset()
	}
	return
}
