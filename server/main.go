package main

import (
	"flag"
	"github.com/gufiejun/easyproxy/server/proxy"
)

const defaultProxyAddr=":8888"

func main(){
	var proxyAddr string
	flag.StringVar(&proxyAddr,"proxy",defaultProxyAddr,"代理地址")
	flag.Parse()
	srv:=proxy.NewRemoteProxy(proxyAddr)
	srv.Serve()
}


