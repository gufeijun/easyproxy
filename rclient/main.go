package main

import (
	"flag"
	"fmt"
	"github.com/gufiejun/easyproxy/rclient/proxy"
)


func main(){
	var remoteAddr string
	flag.StringVar(&remoteAddr,"remote","","具有公网IP主机监听的地址")
	flag.Parse()
	if remoteAddr==""{
		panic("remoteAddr can not be empty")
	}
	fmt.Println("remoteAddr:",remoteAddr)
	s:=proxy.NewIntranetProxy(remoteAddr)
	s.Serve()
}
