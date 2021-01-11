![GitHub](https://img.shields.io/github/license/gufeijun/easyproxy)

# 介绍

easyproxy是一个基于golang的http/https的proxy代理工具，可以实现将http的请求交给一个不具有公网IP的内网主机，让其代为请求。

寒假回家，需要用学校的IP来查阅些资料，遂结合内网穿透原理以及代理机制写了一套软件，只要一台公网服务器以及处在学校的个人PC(可以找在校同学)，就可以白嫖学校的网络，用于实现上知网、激活正版软件等目的。

# 编译

请使用以下方式编译或在release页面下载已编译版本。

```shell
# 在具有公网IP的服务器上
cd server
go build -o proxyServer main.go

# 在处于内网的机器上，这个机器将代理所有http请求
cd rclient
go build -o rclient main.go

# 在需要代理服务的机器上
cd lclient
go build -o lclient main.go
```

# 使用

```shell
# 三者执行顺序：proxyServer、rclient、lclient

# 在具有公网IP的服务器上
proxyServer -proxy ":8888"				  #指定在8888端口监听，默认也为8888

# 在处于内网的机器上
rclient -remote server_address		#例：rclient -remote "175.25.33.33:8888"
#执行成功会返回一个ID号，lclient运行时需要这个ID号

# 在需要代理服务的机器上
lclient -remote server_address -id ID -local 127.0.0.1:1080 
# 如果ID指定为0则指让具有公网IP的服务器进行http代理，否则让对应ID的内网机器进行代理
```

在本机浏览器上安装插件`Proxy SwitchyOmega`或者其他插件，设置http代理为http://127.0.0.1:1080即可，如下。

<a href="https://sm.ms/image/8BVqwSjsfeRFAuJ" target="_blank"><img src="https://i.loli.net/2021/01/11/8BVqwSjsfeRFAuJ.png" ></a>

暂不支持socks5协议。

# 原理

整套软件分为三部分，分别运行在三种不同的机器上。

第一个机器为需要代理请求的主机，比如在校外的个人PC，后续我将称之为主机A。

第二个机器为具有公网IP的主机，其作用是作为一个`tracker`，帮助实现内网穿透，这个机器我后续称为主机B。

第三个机器为最终实际上发起http请求的内网主机，比如是在校内一个机器，后续称之为主机C。

三种程序对应项目的包名如下：

+ **lclient**：运行在主机A上。

+ **server**：运行在主机B上。
+ **rclient**：运行在主机C上。

原理很简单，因为lclient和rclient都不具备公网IP，lclient不能与rclient直接通信，因此需要一个具有公网IP的机器作为中转点。

lclient先拦截所有浏览器的http请求，然后将http请求交给server，server再交给rclient，rclient再真正发出http请求，所以lclient对外显示的IP地址为rclient的IP地址，就实现了嫖学校网络的目标。

# 改进

真正难点是处理好rclient与server端的长连接问题，此处的连接只能由rclient优先发起，如果一开始维持一个长连接池，并对连接池的连接一直复用的话，需要设计一套较为复杂的协议来控制。而且浏览器很多http请求中头部的`connection`字段都是`keep-alive`，就导致会产生很多长连接，一个页面请求完成浏览器并不会释放连接，导致连接池中的长连接一直被占用，导致死锁。

暂时我的解决方案是，rclient与server之间存在两种连接，一种为控制连接，一种为数据连接。控制连接即起了发送心跳包的作用，同时也会可以rclient需要发起数据连接。数据连接起到了发送http请求的作用。

整个过程大概流程：

rclient连接server端，第一次tcp连接注册为控制连接，server会给rclient分配一个ID来区分其他的rclient。接着lclient指定一个ID的rclient作为代理，将http请求转发给server，server读出lclient指定的ID，并通过控制连接给对应ID的rclient发送一个希望其发起数据连接的请求。rclient随后就向server发起数据连接，server就可以把http请求通过数据连接转发给rclient，rclient就可以进行最终的代理。

上述逻辑比较清晰，但弊端也很明显，server每从lclient接收到一个代理请求就必须告知让rclient发起新的连接，再然后在这个连接进行传输，不仅浪费了很多不必要的资源，同时也增加了整个过程的时延。

现在想到的两个改进方案：

+ 维持一个rclient与server端的长连接池，设计一套更复杂的协议进行连接的复用。
+ 维持一个rclient与server端的长连接池，依旧是连接用一个少一个，随后rclient发起新连接补充，虽然依旧有很多资源浪费，但至少可以来一个代理请求时立马有连接可以用，减少时延。依旧要解决浏览器长连接的问题，不然rclient与server端的连接可能无限增长，可以在lclient端将http协议头部的`connction`字段都改成`close`。

# 协议

现有的协议：

| from    | to      | MsgID(4B) | 剩余头部长度(4B) |                                                          |            | 含义                                     |
| ------- | ------- | --------- | ---------------- | -------------------------------------------------------- | ---------- | ---------------------------------------- |
| lcient  | server  | 1         | 4+len(host)      | 指定的内网代理ID，为0代表让server代理(**4B的proxyID**)   | 不定长host | lcient向 server发送一个代理请求。        |
| server  | lclient | 2         | 4                | 响应状态码status(**4B**)                                 | \          | 向lcient返回的响应码                     |
| server  | rclient | 3         | len(host)        | host不定长                                               | \          | server端告诉lclient真实host              |
| rclient | server  | 4         | 4                | proxyID号(**4B**) 0代表注册控制连接 否则代表注册数据连接 | \          | 注册连接                                 |
| server  | rclient | 5         | 4                | proxyID号(**4B**)                                        | \          | 给控制连接写入分配的ID                   |
| server  | rclient | 6         | 4                | 需要rclient发起多少数据连接                              | \          | 请求更多数据连接                         |
| rclient | server  | 7         | 4                | 心跳(**4B**)，1代表在线 0代表下线                        | \          | 心跳包，服务端被动接受，通过控制连接发送 |

头部8B固定，前4B为ID，后4B未剩余头部的长度。