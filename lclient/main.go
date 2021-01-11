package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gufiejun/easyproxy/lclient/proxy"
)


func main(){
	var remoteAddr string
	var localAddr string
	//需要指定在内网内提供代理的主机的ID号，这个ID号是代理主机向具有公网IP主机注册时分配的
	var proxyID int
	flag.StringVar(&remoteAddr,"remote","","远程代理服务器的监听地址")
	flag.StringVar(&localAddr,"local","127.0.0.1:1080","本地代理监听地址")
	//如果为0则指定由具有公网的主机进行proxy代理
	flag.IntVar(&proxyID,"id",0,"在内网内提供proxy服务的主机ID")

	flag.Parse()
	if remoteAddr==""{
		fmt.Println("remoteAddr must be specific")
		return
	}
	fmt.Println("提供proxy的内网主机ID:",proxyID)
	lProxy:=proxy.NewLocalProxy(remoteAddr,localAddr,uint32(proxyID))
	log.Fatal(lProxy.Serve())
}


