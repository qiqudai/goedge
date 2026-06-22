package main

import (
	"fmt"
	"os"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: lookup_ip <xdb> <ip>")
		os.Exit(1)
	}
	searcher, err := xdb.NewWithFileOnly(xdb.IPv4, os.Args[1])
	if err != nil {
		panic(err)
	}
	defer searcher.Close()
	region, err := searcher.SearchByStr(os.Args[2])
	fmt.Printf("region=%q err=%v\n", region, err)
}
