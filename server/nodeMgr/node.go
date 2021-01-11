package nodeMgr

import (
	"io"
	"net"
	"sync"

	"github.com/gufiejun/easyproxy"
)

const defaultChanCap=100
var packet = easyproxy.NewPacket()

//每一个Node代表一个内网内的proxy机器
//每个Node都会和有公网IP的中心主机维持多个长连接
type Node struct {
	//ControlConn用于给Node发送命令
	controlConn net.Conn
	//DataConns用于数据传输
	dataConnCh chan net.Conn
	mu sync.Mutex
	id uint32
	nodeMgr *NodeMgr
	once sync.Once
}

func NewNode(controlConn net.Conn,nodeMgr *NodeMgr)*Node{
	node:=&Node{
		dataConnCh: make(chan net.Conn,defaultChanCap),
		controlConn: controlConn,
		nodeMgr: nodeMgr,
	}
	go node.heartBeat()
	return node
}

//被动接受心跳包，如果read出现err则回收节点资源
func (n *Node)heartBeat(){
	defer n.Close()
	buf:=make([]byte,12)
	for{
		_,err:=io.ReadFull(n.controlConn,buf)
		if err!=nil{
			return
		}
	}
}

func (n *Node)PutConn(conn net.Conn){
	n.dataConnCh<-conn
}

func (n *Node)GetConn()<-chan net.Conn{
	n.mu.Lock()
	if err:=n.NotifyMoreConn(1);err!=nil{
		n.Close()
	}
	n.mu.Unlock()
	return n.dataConnCh
}

func (n *Node)Close(){
	n.once.Do(func() {
		n.nodeMgr.DeleteNode(n)
		n.controlConn.Close()
		close(n.dataConnCh)
		for conn:=range n.dataConnCh{
			conn.Close()
		}
	})
}

func (n *Node)GetID()uint32{
	return n.id
}

func (n *Node)NotifyMoreConn(num uint32)error{
	buf:=make([]byte,12)
	if err:=packet.Write(buf,easyproxy.MSG6,4,num);err!=nil{
		return err
	}
	_,err:=n.controlConn.Write(buf)
	return err
}