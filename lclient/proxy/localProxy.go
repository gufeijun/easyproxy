package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"sync"

	"github.com/gufiejun/easyproxy"
)

const bufSize = 256

var pool = sync.Pool{
	New: func() interface{} {
		return make([]byte, bufSize)
	},
}

var packet = easyproxy.NewPacket()

/*
LocalProxy任务：
	监听来自浏览器的连接，完成与具有公网IP的中心服务器的握手，
	并进行浏览器与中心服务器之间数据的转发
*/
type LocalProxy struct {
	ListeningAddr   string
	RemoteProxyAddr string
	ProxyID         uint32
}

func NewLocalProxy(remote, local string, proxyID uint32) *LocalProxy {
	return &LocalProxy{
		ListeningAddr:   local,
		RemoteProxyAddr: remote,
		ProxyID:         proxyID,
	}
}

func (l *LocalProxy) Serve() error {
	TCPAddr, err := net.ResolveTCPAddr("tcp", l.ListeningAddr)
	if err != nil {
		return err
	}
	listener, err := net.ListenTCP("tcp", TCPAddr)
	if err != nil {
		return err
	}
	fmt.Println("proxy listening on ", l.ListeningAddr)
	fmt.Println("remote proxy addr", l.RemoteProxyAddr)
	defer listener.Close()
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Println("accept TCPConn error", err)
			continue
		}
		go l.handleConn(conn)
	}
}

func (l *LocalProxy) handleConn(conn net.Conn) {
	defer func() {
		if err:=recover();err!=nil{
			fmt.Println("panic recovered")
		}
	}()
	defer conn.Close()
	bf := pool.Get().([]byte)
	defer pool.Put(bf)
	//http和https请求头格式如下
	/*
		HTTPS:
		CONNECT www.baidu.com:443 HTTP/1.1
		Host: www.baidu.com:443
		Proxy-Connection: keep-alive

		HTTP:
		GET http://www.hahasdhfjas.com/ HTTP/1.1
		Host: www.hahasdhfjas.com
		Proxy-Connection: keep-alive
	*/

	//step1 解析出http或者https请求的真实地址
	n, err := conn.Read(bf)
	if err != nil {
		return
	}
	//读取第一行
	lineEndIndex := bytes.IndexByte(bf[:n], '\n')
	//解析出method和host
	method, host, err := parse(bf[:lineEndIndex])
	if err != nil {
		fmt.Println("parse method and host err:", err)
		return
	}

	//step2 连接远端具有公网IP的服务器
	remoteConn, err := net.Dial("tcp", l.RemoteProxyAddr)
	if err != nil {
		fmt.Println("can not reach to remote server:", l.RemoteProxyAddr)
		return
	}
	defer remoteConn.Close()

	//step3 向远端服务器发送希望处理自己请求的代理人ID以及真实请求的host地址
	buf := make([]byte, 12+len(host))
	//消息的ID号为1
	if err := packet.Write(buf, easyproxy.MSG1, uint32(len(host)+4),l.ProxyID); err != nil {
		panic(err)
	}
	copy(buf[12:], host)

	length := 0
	for length != len(buf) {
		n, err := remoteConn.Write(buf[length:])
		if err != nil {
			return
		}
		length += n
	}

	//step4 接受来自远端服务器的应答信息，服务器需要先判断指定的ID是否存在或者可用，然后返回状态码
	n, err = io.ReadFull(remoteConn, buf[:12])
	if err != nil || n != 12 {
		return
	}
	msgID := packet.Read(buf[:4])
	status := packet.Read(buf[8:12])
	if msgID != easyproxy.MSG2 {
		log.Panic("错误响应码")
	}
	if status == easyproxy.STATUS_TIMEOUT {
		log.Println("获取连接超时")
		return
	}
	if status == easyproxy.STATUS_UNREGISTER_MACHINE {
		log.Panicf("指定ID:%d的内网机器并未注册", l.ProxyID)
	}
	if status != easyproxy.STATUS_SUCCESS {
		log.Panic("未定义的状态码")
	}

	//step5 如果是http则需要将先前读出的头部发送给服务端
	//https则只需要给客户端返回一个连接建立成功的信息
	if method == "CONNECT" {
		_, err := io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
		if err != nil {
			return
		}
	} else {
		_, err := remoteConn.Write(bf[:n])
		if err != nil {
			return
		}
	}

	//step6 握手已经成功，只需要两端数据转发即可
	go io.Copy(conn, remoteConn)
	io.Copy(remoteConn, conn)
}

func parse(line []byte) (method string, host string, err error) {
	if _, err = fmt.Sscanf(string(line), "%s%s", &method, &host); err != nil {
		fmt.Println("read line err")
		return
	}
	//如果为https请求，直接返回无需再次解析
	if method == "CONNECT" {
		return
	}
	//否则对http的host进行再次解析,例:http://www.example.com/
	URL, err := url.Parse(host)
	if err != nil {
		return
	}
	if URL.Port() == "" {
		host = fmt.Sprintf("%s:80", URL.Host)
	} else {
		host = fmt.Sprintf("%s:%s", URL.Host, URL.Port())
	}
	return
}