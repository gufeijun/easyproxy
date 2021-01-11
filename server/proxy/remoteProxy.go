package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/gufiejun/easyproxy"
	"github.com/gufiejun/easyproxy/server/nodeMgr"
)

const defaultGetConnTimeOut = 30 * time.Second

var packet = easyproxy.NewPacket()

/*
RemoteProxy任务：
	接受来自LocalProxy的握手信息，管理实际进行代理的内网主机的TCP长连接，
	进行上述两者连接的数据转发。
*/
type RemoteProxy struct {
	//提供代理服务的监听地址
	ProxyAddr string
	//管理所有的内网主机
	nodeMgr *nodeMgr.NodeMgr
	funcMap map[uint32]func(conn net.Conn, headLen uint32)
}

type handleFunc func(net.Conn)

func NewRemoteProxy(ProxyAddr string) *RemoteProxy {
	p := &RemoteProxy{
		ProxyAddr:    ProxyAddr,
		nodeMgr:      nodeMgr.NewNodeMgr(),
	}
	funcMap := map[uint32]func(net.Conn, uint32){
		easyproxy.MSG1: p.handleMsg1Conn,
		easyproxy.MSG4: p.handleMsg4Conn,
	}
	p.funcMap = funcMap
	return p
}

func (p *RemoteProxy) Serve() {
	TCPAddr, err := net.ResolveTCPAddr("tcp",p.ProxyAddr)
	if err != nil {
		panic(err)
	}
	listener, err := net.ListenTCP("tcp", TCPAddr)
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening on ",p.ProxyAddr)
	defer listener.Close()
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}
		go p.handleConn(conn)
	}
}

//处理localProxy(即lclient)发来的请求
func (p *RemoteProxy) handleConn(clientConn net.Conn) {
	//step1 读出消息类型，以及后续头部的长度
	buf := make([]byte, 8)
	_, err := io.ReadFull(clientConn, buf)
	if err != nil {
		log.Println(err)
		clientConn.Close()
		return
	}
	msgID := packet.Read(buf[:4])
	length := packet.Read(buf[4:8])

	f, ok := p.funcMap[msgID]
	if !ok {
		clientConn.Close()
		return
	}
	f(clientConn, length)
}

//由lClient发起的连接
func (p *RemoteProxy) handleMsg1Conn(conn net.Conn, length uint32) {
	defer conn.Close()
	if length <= 4 {
		return
	}
	//step1 读出ID号以及host
	buf := make([]byte, length)
	n, err := io.ReadFull(conn, buf)
	if err != nil || uint32(n) != length {
		return
	}
	proxyID := packet.Read(buf[:4])
	host := buf[4:]

	//step2
	//如果proxyID为0代表让此server作为代理服务器
	if proxyID == 0 {
		hostConn, err := net.Dial("tcp", string(host))
		if err != nil {
			return
		}
		defer hostConn.Close()
		err = responseToLClient(conn, easyproxy.STATUS_SUCCESS)
		if err != nil {
			return
		}
		go io.Copy(hostConn, conn)
		io.Copy(conn, hostConn)
		return
	}
	//如果不为0则需要将代理请求继续转交给已经注册的内网机器
	node, ok := p.nodeMgr.GetNode(proxyID)
	if !ok {
		responseToLClient(conn, easyproxy.STATUS_UNREGISTER_MACHINE)
		return
	}
	timer := time.NewTimer(defaultGetConnTimeOut)
	select {
	//如果拿连接超时直接返回
	case <-timer.C:
		responseToLClient(conn, easyproxy.STATUS_TIMEOUT)
		return
	case rClientConn, ok := <-node.GetConn():
		if !ok {
			return
		}
		defer rClientConn.Close()
		if err:=responseToLClient(conn,easyproxy.STATUS_SUCCESS);err!=nil{
			return
		}
		//向rClient写入host地址
		buf := make([]byte, 8+len(host))
		packet.Write(buf, easyproxy.MSG3, uint32(len(host)))
		copy(buf[8:], host)

		if  _, err := rClientConn.Write(buf);err!=nil {
			return
		}
		go io.Copy(rClientConn,conn)
		io.Copy(conn,rClientConn)
	}
}

//由rClient发起的连接
func (p *RemoteProxy)handleMsg4Conn(rClientConn net.Conn,length uint32){
	buf:=make([]byte,4)
	_,err:=io.ReadFull(rClientConn,buf)
	if err!=nil{
		rClientConn.Close()
		return
	}
	proxyID:=packet.Read(buf)
	//如果proxyID为0代表此连接为一个控制连接，代表内网节点向公网服务器注册。
	//为该内网节点分配一个ID，随后内网节点就会以该ID注册数据连接
	if proxyID==0{
		node := nodeMgr.NewNode(rClientConn, p.nodeMgr)
		p.nodeMgr.AddNode(node)
		buf:=make([]byte,12)
		packet.Write(buf,easyproxy.MSG5,4,node.GetID())
		if _,err:=rClientConn.Write(buf);err!=nil{
			node.Close()
			return
		}
		return
	}
	//如果ID不为1代表为此连接为数据连接，代表向对应ID的节点注册数据连接
	node, ok := p.nodeMgr.GetNode(proxyID)
	//如果指定的ID都不存在则直接退出
	if !ok {
		rClientConn.Close()
		return
	}
	//将数据连接放入node中保存
	node.PutConn(rClientConn)
}

func responseToLClient(conn net.Conn, status uint32) error {
	buf := make([]byte, 12)
	packet.Write(buf, 2, 4, status)
	if _, err := conn.Write(buf); err != nil {
		return err
	}
	return nil
}