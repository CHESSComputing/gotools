package main

import (
	"fmt"

	"github.com/CHESSComputing/golib/utils"
)

func main() {
	fmt.Println("List of MAC addresses:")
	for _, mac := range utils.MacAddr() {
		fmt.Printf("%+v\n", mac)
	}
	fmt.Println("List of IP addresses:")
	for _, addr := range utils.IpAddr() {
		fmt.Println(addr)
	}
}
