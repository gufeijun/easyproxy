package proxy

import (
	"fmt"
	"github.com/gufiejun/easyproxy"
	"io"
	"net"
	"time"
)

/*
IntranetProxy的任务：
	向具有公网IP的服务器注册；接受代理任务
*/
type IntranetProxy struct {
	RemoteAddr     string
	ControlConn    net.Conn
	//自己的ID号，用于区分其他机器
	ID uint32
}

var packet = easyproxy.NewPacket()

func NewIntranetProxy(RemoteAddr string) *IntranetProxy {
	return &IntranetProxy{
		RemoteAddr: RemoteAddr,
	}
}

func (p *IntranetProxy) Serve() {
	p.StartControlConn()
}

func (p *IntranetProxy) StartControlConn() {
	conn, err := net.Dial("tcp", p.RemoteAddr)
	if err != nil {
		panic(err)
	}
	//发送0代表申请ID,并将该连接注册为控制连接
	buf:=make([]byte,12)
	if err:=packet.Write(buf,easyproxy.MSG4,4,0);err!=nil{
		conn.Close()
		return
	}

	n,err:=conn.Write(buf)
	if err!=nil||n!=12{
		panic("send request error")
	}

	n,err=io.ReadFull(conn,buf)
	if err!=nil||n!=12{
		panic("read response error")
	}
	msgID,proxyID:=packet.Read(buf[:4]),packet.Read(buf[8:12])
	if msgID!=easyproxy.MSG5{
		conn.Close()
		return
	}

	p.ControlConn = conn
	p.ID = proxyID
	fmt.Printf("连接服务成功，自己的ID为%d\n", p.ID)

	go p.heartBeat()
	//等待服务端通过控制连接发送请求
	p.waitingForCommand()
}

func (p *IntranetProxy)heartBeat(){
	buf:=make([]byte,12)
	packet.Write(buf,easyproxy.MSG7,4,1)
	for{
		time.Sleep(30*time.Second)
		_,err:=p.ControlConn.Write(buf)
		if err!=nil{
			return
		}
	}
}

func (p *IntranetProxy)waitingForCommand(){
	buf:=make([]byte,12)
	for{
		_,err:=io.ReadFull(p.ControlConn,buf)
		if err!=nil{
			continue
		}
		msgID,num:=packet.Read(buf[:4]),packet.Read(buf[8:12])
		if msgID!=easyproxy.MSG6{
			continue
		}
		for i:=0;i<int(num);i++{
			go p.startDataConn()
		}
	}
}

func (p *IntranetProxy) startDataConn() {
	conn, err := net.Dial("tcp", p.RemoteAddr)
	if err != nil {
		panic("can not reach to remote server")
		return
	}
	defer conn.Close()

	buf:=make([]byte,12)
	packet.Write(buf,easyproxy.MSG4,4,p.ID)
	_,err=conn.Write(buf)
	if err!=nil{
		return
	}

	//读取ID为3的消息
	_,err=io.ReadFull(conn,buf[:8])
	if err!=nil{
		return
	}
	msgID:=packet.Read(buf[:4])
	if msgID!=easyproxy.MSG3{
		return
	}
	hostLen:=packet.Read(buf[4:8])
	if len(buf)<int(hostLen){
		buf=make([]byte,hostLen)
	}

	if _,err=io.ReadFull(conn,buf[:hostLen]);err!=nil{
		return
	}

	host:=string(buf[:hostLen])
	hostConn, err := net.Dial("tcp",host )
	if err != nil {
		fmt.Println("dial host err,host=", host)
		return
	}
	go io.Copy(conn, hostConn)
	io.Copy(hostConn, conn)
}
